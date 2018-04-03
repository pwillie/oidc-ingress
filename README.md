# oidc-ingress

A webhook authentication service using OIDC and cookies

Motivation for creating this service is to easily add OIDC authentication to any
service running behind an Nginx Ingress controller in Kubernetes.  By using cookies
there is no need for client side changes and any legacy system/service can be authenticated.

## Kubernetes Nginx Ingress OIDC sequence diagram

![OIDC Sequence Diagram](https://github.com/pwillie/oidc-ingress/blob/master/images/sequence.png?raw=true "OIDC Sequence Diagram")

Created using: *https://sequencediagram.org/*

## Configuration

| Env Var  | CMD line arg | Default Value | Notes |
|----------|--------------|---------------|-------|
| CLIENTS  | -clients     | -             | OIDC clients config expressed in yaml (see below) |
| LISTEN   | -listen      | :8000         | Web server listen address |
| INTERNAL | -internal    | :9000         | Internal listen address for healthz and metrics endpoints |
| VERSION  | -version     | -             | When set will print version and exit |

## Clients

Clients env var (or cmd line arg) is a YAML formated string.  For example:
```
- provider: https://oauth.provider.url/
  clientid: client_id
  clientsecret: client_secret
  noredirect: false (default: false)
  scopes: (default: - openid)
    - openid
    - email
    - profile
```

*note:* `noredirect` will suppress the `?rd={redirect url}` from the path.  Handy for Azure AD as querystring is stripped anyway and redirect url must match exactly.

## Building

```console
$ make build
$ ./bin/oidc-ingress
```

## Testing

```
$ make test
```
