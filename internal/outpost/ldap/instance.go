package ldap

import (
	"crypto/tls"
	"fmt"
	"strings"
	"sync"

	"github.com/go-openapi/strfmt"
	"github.com/nmcclain/ldap"
	log "github.com/sirupsen/logrus"
	"goauthentik.io/api"
	"goauthentik.io/internal/constants"
	"goauthentik.io/internal/outpost/ldap/bind"
	ldapConstants "goauthentik.io/internal/outpost/ldap/constants"
	"goauthentik.io/internal/outpost/ldap/flags"
	"goauthentik.io/internal/outpost/ldap/search"
	"goauthentik.io/internal/outpost/ldap/utils"
)

type ProviderInstance struct {
	BaseDN         string
	UserDN         string
	VirtualGroupDN string
	GroupDN        string

	searcher search.Searcher
	binder   bind.Binder

	appSlug  string
	flowSlug string
	s        *LDAPServer
	log      *log.Entry

	tlsServerName       *string
	cert                *tls.Certificate
	outpostName         string
	outpostPk           int32
	searchAllowedGroups []*strfmt.UUID
	boundUsersMutex     sync.RWMutex
	boundUsers          map[string]flags.UserFlags

	uidStartNumber int32
	gidStartNumber int32
}

func (pi *ProviderInstance) GetAPIClient() *api.APIClient {
	return pi.s.ac.Client
}

func (pi *ProviderInstance) GetBaseDN() string {
	return pi.BaseDN
}

func (pi *ProviderInstance) GetBaseGroupDN() string {
	return pi.GroupDN
}

func (pi *ProviderInstance) GetBaseVirtualGroupDN() string {
	return pi.VirtualGroupDN
}

func (pi *ProviderInstance) GetBaseUserDN() string {
	return pi.UserDN
}

func (pi *ProviderInstance) GetOutpostName() string {
	return pi.outpostName
}

func (pi *ProviderInstance) GetFlags(dn string) (flags.UserFlags, bool) {
	pi.boundUsersMutex.RLock()
	flags, ok := pi.boundUsers[dn]
	pi.boundUsersMutex.RUnlock()
	return flags, ok
}

func (pi *ProviderInstance) SetFlags(dn string, flag flags.UserFlags) {
	pi.boundUsersMutex.Lock()
	pi.boundUsers[dn] = flag
	pi.boundUsersMutex.Unlock()
}

func (pi *ProviderInstance) GetAppSlug() string {
	return pi.appSlug
}

func (pi *ProviderInstance) GetFlowSlug() string {
	return pi.flowSlug
}

func (pi *ProviderInstance) GetSearchAllowedGroups() []*strfmt.UUID {
	return pi.searchAllowedGroups
}

func (pi *ProviderInstance) GetBaseEntry() *ldap.Entry {
	return &ldap.Entry{
		DN: pi.GetBaseDN(),
		Attributes: []*ldap.EntryAttribute{
			{
				Name:   "distinguishedName",
				Values: []string{pi.GetBaseDN()},
			},
			{
				Name:   "objectClass",
				Values: []string{ldapConstants.OCTop, ldapConstants.OCDomain},
			},
			{
				Name:   "supportedLDAPVersion",
				Values: []string{"3"},
			},
			{
				Name: "namingContexts",
				Values: []string{
					pi.GetBaseDN(),
					pi.GetBaseUserDN(),
					pi.GetBaseGroupDN(),
					pi.GetBaseVirtualGroupDN(),
				},
			},
			{
				Name:   "vendorName",
				Values: []string{"goauthentik.io"},
			},
			{
				Name:   "vendorVersion",
				Values: []string{fmt.Sprintf("authentik LDAP Outpost Version %s", constants.FullVersion())},
			},
		},
	}
}

func (pi *ProviderInstance) GetNeededObjects(scope int, baseDN string, filterOC string) (bool, bool) {
	needUsers := false
	needGroups := false

	// We only want to load users/groups if we're actually going to be asked
	// for at least one user or group based on the search's base DN and scope.
	//
	// If our requested base DN doesn't match any of the container DNs, then
	// we're probably loading a user or group. If it does, then make sure our
	// scope will eventually take us to users or groups.
	if (strings.EqualFold(baseDN, pi.BaseDN) || utils.HasSuffixNoCase(baseDN, pi.UserDN)) && utils.IncludeObjectClass(filterOC, ldapConstants.GetUserOCs()) {
		if baseDN != pi.UserDN && baseDN != pi.BaseDN ||
			strings.EqualFold(baseDN, pi.BaseDN) && scope > 1 ||
			strings.EqualFold(baseDN, pi.UserDN) && scope > 0 {
			needUsers = true
		}
	}

	if (strings.EqualFold(baseDN, pi.BaseDN) || utils.HasSuffixNoCase(baseDN, pi.GroupDN)) && utils.IncludeObjectClass(filterOC, ldapConstants.GetGroupOCs()) {
		if baseDN != pi.GroupDN && baseDN != pi.BaseDN ||
			strings.EqualFold(baseDN, pi.BaseDN) && scope > 1 ||
			strings.EqualFold(baseDN, pi.GroupDN) && scope > 0 {
			needGroups = true
		}
	}

	if (strings.EqualFold(baseDN, pi.BaseDN) || utils.HasSuffixNoCase(baseDN, pi.VirtualGroupDN)) && utils.IncludeObjectClass(filterOC, ldapConstants.GetVirtualGroupOCs()) {
		if baseDN != pi.VirtualGroupDN && baseDN != pi.BaseDN ||
			strings.EqualFold(baseDN, pi.BaseDN) && scope > 1 ||
			strings.EqualFold(baseDN, pi.VirtualGroupDN) && scope > 0 {
			needUsers = true
		}
	}

	return needUsers, needGroups
}
