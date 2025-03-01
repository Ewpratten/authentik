---
title: Docker
---

The docker integration will automatically deploy and manage outpost containers using the Docker HTTP API.

This integration has the advantage over manual deployments of automatic updates (whenever authentik is updated, it updates the outposts), and authentik can (in a future version) automatically rotate the token that the outpost uses to communicate with the core authentik server.

The following outpost settings are used:

- `object_naming_template`: Configures how the container is called
- `container_image`: Optionally overwrites the standard container image (see [Configuration](../../installation/configuration.md) to configure the global default)
- `docker_network`: The docker network the container should be added to. This needs to be modified if you plan to connect to authentik using the internal hostname.
- `docker_map_ports`: Enable/disable the mapping of ports. When using a proxy outpost with traefik for example, you might not want to bind ports as they are routed through traefik.
- `docker_labels`: Optional additional labels that can be applied to the container.

The container is created with the following hardcoded properties:

- Labels

    - `io.goauthentik.outpost-uuid`: Used by authentik to identify the container, and to allow for name changes.

    Additionally, the proxy outposts have the following extra labels to add themselves into traefik automatically.

    - `traefik.enable`: "true"
    - `traefik.http.routers.ak-outpost-<outpost-id>-router.rule`: `Host(...)`
    - `traefik.http.routers.ak-outpost-<outpost-id>-router.service`: `ak-outpost-<outpost-id>-service`
    - `traefik.http.routers.ak-outpost-<outpost-id>-router.tls`: "true"
    - `traefik.http.services.ak-outpost-<outpost-id>-service.loadbalancer.healthcheck.path`: "/akprox/ping"
    - `traefik.http.services.ak-outpost-<outpost-id>-service.loadbalancer.healthcheck.port`: "9300"
    - `traefik.http.services.ak-outpost-<outpost-id>-service.loadbalancer.server.port`: "9000"

## Permissions

To minimise the potential risks of mapping the docker socket into a container/giving an application access to the docker API, many people use Projects like [docker-socket-proxy](https://github.com/Tecnativa/docker-socket-proxy). authentik requires these permissions from the docker API:

- Images/Pull: authentik tries to pre-pull the custom image if one is configured, otherwise falling back to the default image.
- Containers/Read: Gather infos about currently running container
- Containers/Create: Create new containers
- Containers/Kill: Cleanup during upgrades
- Containers/Remove: Removal of outposts

## Remote hosts (TLS)

To connect remote hosts, you can follow this Guide from Docker [Use TLS (HTTPS) to protect the Docker daemon socket](https://docs.docker.com/engine/security/protect-access/#use-tls-https-to-protect-the-docker-daemon-socket) to configure Docker.

Afterwards, create two Certificate-keypairs in authentik:

- `Docker CA`, with the contents of `~/.docker/ca.pem` as Certificate
- `Docker Cert`, with the contents of `~/.docker/cert.pem` as Certificate and `~/.docker/key.pem` as Private key.

Create an integration with `Docker CA` as *TLS Verification Certificate* and `Docker Cert` as *TLS Authentication Certificate*.

## Remote hosts (SSH)

Starting with authentik 2021.12.5, you can connect to remote docker hosts using SSH. To configure this, create a new SSH keypair using these commands:

```
# Generate the keypair itself, using RSA keys in the PEM format
ssh-keygen -t rsa -f authentik  -N "" -m pem
# Generate a certificate from the private key, required by authentik.
# The values that openssl prompts you for are not relevant
openssl req -x509 -sha256 -nodes -days 365 -out certificate.pem -key authentik
```

You'll end up with three files:

- `authentik.pub` is the public key, this should be added to the `~/.ssh/authorized_keys` file on the target host and user.
- `authentik` is the private key, which should be imported into a Keypair in authentik.
- `certificate.pem` is the matching certificate for the keypair above.

Modify/create a new Docker integration, and set your *Docker URL* to `ssh://hostname`, and select the keypair you created above as *TLS Authentication Certificate/SSH Keypair*.

The *Docker URL* field include a user, if none is specified authentik connects with the user `authentik`.
