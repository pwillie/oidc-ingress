# oidc-ingress

A webhook authentication service using OIDC and cookies

Motivation for creating this service is to easily add OIDC authentication to any
service running behind an Nginx Ingress controller in Kubernetes.  By using cookies
there is no need for client side changes and any legacy system/service can be authenticated.

## Kubernetes Nginx Ingress OIDC sequence diagram

![OIDC Sequence Diagram](https://github.com/pwillie/oidc-ingress/blob/master/images/sequence.png?raw=true "OIDC Sequence Diagram")

Created using: *https://sequencediagram.org/*

## Getting started

This project requires Go to be installed. On OS X with Homebrew you can just run `brew install go`.

Running it then should be as simple as:

```console
$ make build
$ ./bin/oidc-ingress
```

### Testing

```
$ make test
```
