package application

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"goauthentik.io/api"
	"goauthentik.io/internal/constants"
)

func (a *Application) addHeaders(headers http.Header, c *Claims) {
	// https://goauthentik.io/docs/providers/proxy/proxy

	headers.Set("X-authentik-username", c.PreferredUsername)
	headers.Set("X-authentik-groups", strings.Join(c.Groups, "|"))
	headers.Set("X-authentik-email", c.Email)
	headers.Set("X-authentik-name", c.Name)
	headers.Set("X-authentik-uid", c.Sub)
	headers.Set("X-authentik-jwt", c.RawToken)

	// System headers
	headers.Set("X-authentik-meta-jwks", a.endpint.JwksUri)
	headers.Set("X-authentik-meta-outpost", a.outpostName)
	headers.Set("X-authentik-meta-provider", a.proxyConfig.Name)
	headers.Set("X-authentik-meta-app", a.proxyConfig.AssignedApplicationSlug)
	headers.Set("X-authentik-meta-version", constants.OutpostUserAgent())

	userAttributes := c.Proxy.UserAttributes
	// Attempt to set basic auth based on user's attributes
	if *a.proxyConfig.BasicAuthEnabled {
		var ok bool
		var password string
		if password, ok = userAttributes[*a.proxyConfig.BasicAuthPasswordAttribute].(string); !ok {
			password = ""
		}
		// Check if we should use email or a custom attribute as username
		var username string
		if username, ok = userAttributes[*a.proxyConfig.BasicAuthUserAttribute].(string); !ok {
			username = c.Email
		}
		authVal := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
		a.log.WithField("username", username).Trace("setting http basic auth")
		headers.Set("Authorization", fmt.Sprintf("Basic %s", authVal))
	}
	// Check if user has additional headers set that we should sent
	if additionalHeaders, ok := userAttributes["additionalHeaders"].(map[string]interface{}); ok {
		a.log.WithField("headers", additionalHeaders).Trace("setting additional headers")
		if additionalHeaders == nil {
			return
		}
		for key, value := range additionalHeaders {
			headers.Set(key, toString(value))
		}
	}
}

// getTraefikForwardUrl See https://doc.traefik.io/traefik/middlewares/forwardauth/
func (a *Application) getTraefikForwardUrl(r *http.Request) *url.URL {
	u, err := url.Parse(fmt.Sprintf(
		"%s://%s%s",
		r.Header.Get("X-Forwarded-Proto"),
		r.Header.Get("X-Forwarded-Host"),
		r.Header.Get("X-Forwarded-Uri"),
	))
	if err != nil {
		a.log.WithError(err).Warning("Failed to parse URL from traefik")
		return r.URL
	}
	a.log.WithField("url", u.String()).Trace("traefik forwarded url")
	return u
}

// getNginxForwardUrl See https://github.com/kubernetes/ingress-nginx/blob/main/rootfs/etc/nginx/template/nginx.tmpl
func (a *Application) getNginxForwardUrl(r *http.Request) *url.URL {
	h := r.Header.Get("X-Original-URL")
	if len(h) < 1 {
		a.log.WithError(errors.New("blank URL")).Warning("blank URL")
		return r.URL
	}
	u, err := url.Parse(h)
	if err != nil {
		a.log.WithError(err).Warning("failed to parse URL from nginx")
		return r.URL
	}
	a.log.WithField("url", u.String()).Trace("nginx forwarded url")
	return u
}

func (a *Application) IsAllowlisted(u *url.URL) bool {
	for _, ur := range a.UnauthenticatedRegex {
		var testString string
		if a.Mode() == api.PROXYMODE_PROXY || a.Mode() == api.PROXYMODE_FORWARD_SINGLE {
			testString = u.Path
		} else {
			testString = u.String()
		}
		a.log.WithField("regex", u.String()).WithField("url", testString).Trace("Matching URL against allow list")
		if ur.MatchString(testString) {
			return true
		}
	}
	return false
}
