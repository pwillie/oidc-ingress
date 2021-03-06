package handlers

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	oidc "github.com/coreos/go-oidc"
	"github.com/ghodss/yaml"
	"github.com/go-chi/chi"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
)

type (
	OidcClient struct {
		Provider     *oidc.Provider
		ClientID     string
		ClientSecret string
		NoRedirect   bool
		Scopes       []string
	}
	Oidc struct {
		clients map[string]*OidcClient
	}
)

const cookieName = "jwt"

var (
	state = "foobar" // TODO: Don't do this in production.
)

func NewOidcHandler(config string) (*Oidc, error) {
	var clientConfigs []struct {
		Provider     string
		ClientID     string
		ClientSecret string
		NoRedirect   bool
		Scopes       []string
	}
	err := yaml.Unmarshal([]byte(config), &clientConfigs)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to parse OIDC client config")
	}

	// Initialize each unique provider
	providers := make(map[string]*oidc.Provider)
	clients := make(map[string]*OidcClient)

	for _, c := range clientConfigs {
		if len(c.Scopes) == 0 {
			c.Scopes = []string{oidc.ScopeOpenID}
		}
		_, ok := providers[c.Provider]
		if !ok {
			// logger.Info(fmt.Sprintf("Initialising OIDC discovery endpoint: %s", c.Provider))
			providers[c.Provider], err = oidc.NewProvider(context.Background(), c.Provider)
			if err != nil {
				return nil, errors.Wrap(err, "Unable to initialise provider")
			}
		}
		clients[c.ClientID] = &OidcClient{
			Provider:     providers[c.Provider],
			ClientID:     c.ClientID,
			ClientSecret: c.ClientSecret,
			NoRedirect:   c.NoRedirect,
			Scopes:       c.Scopes,
		}
		// logger.Info(fmt.Sprintf("ClientID: %v, Provider: %v\n", c.ClientID, c.Provider))
	}
	if len(clients) == 0 {
		return nil, errors.New("No OIDC clients configured")
	}
	return &Oidc{clients}, nil
}

// helpers

func (c OidcClient) verifyToken(token string) error {
	idTokenVerifier := c.Provider.Verifier(
		&oidc.Config{ClientID: c.ClientID, SupportedSigningAlgs: []string{"RS256"}},
	)
	_, err := idTokenVerifier.Verify(context.Background(), token)
	return err
}

func (c OidcClient) redirectURL(r *http.Request) string {
	host := r.Host
	if h := r.Header.Get("X-Original-Url"); h != "" {
		u, err := url.Parse(h)
		if err == nil {
			host = u.Hostname()
		}
	}
	var rd string
	if !c.NoRedirect {
		rd = r.URL.Query().Get("rd")
		if rd != "" {
			rd = "?rd=" + rd
		}
	}
	return fmt.Sprintf("https://%v/auth/callback/%v%v",
		host, c.ClientID, rd,
	)
}

func (c OidcClient) oAuth2Config(redirect string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     c.ClientID,
		ClientSecret: c.ClientSecret,
		Endpoint:     c.Provider.Endpoint(),
		RedirectURL:  redirect,
		Scopes:       c.Scopes,
	}
}

// Handlers

func (o Oidc) VerifyHandler(w http.ResponseWriter, r *http.Request) {
	clientID := chi.URLParam(r, "clientid")
	if config, ok := o.clients[clientID]; ok {
		token, err := r.Cookie(cookieName)
		if token != nil {
			config.verifyToken(token.Value)
			if err == nil {
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	w.WriteHeader(http.StatusForbidden)
}

func (o Oidc) SigninHandler(w http.ResponseWriter, r *http.Request) {
	clientID := chi.URLParam(r, "clientid")
	if config, ok := o.clients[clientID]; ok {
		token, err := r.Cookie(cookieName)
		if token != nil {
			config.verifyToken(token.Value)
			if err == nil {
				if r.URL.Query().Get("rd") != "" {
					http.Redirect(w, r, r.URL.Query().Get("rd"), http.StatusFound)
					return
				}
				w.WriteHeader(http.StatusOK)
				return
			}
		}
		http.Redirect(w, r, config.oAuth2Config(config.redirectURL(r)).AuthCodeURL(state), http.StatusFound)
		return
	}
	w.WriteHeader(http.StatusInternalServerError)
	fmt.Fprint(w, "Configuration error")
}

func (o Oidc) CallbackHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("state") != state {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "Invalid state")
	}

	clientID := chi.URLParam(r, "clientid")
	if config, ok := o.clients[clientID]; ok {
		oauth2Token, err := config.oAuth2Config(config.redirectURL(r)).Exchange(context.Background(), r.URL.Query().Get("code"))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "Failed to exchange token: %s", err.Error())
			return
		}

		rawIDToken, ok := oauth2Token.Extra("id_token").(string)
		if !ok {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, "No id_token field in oauth2 token.")
			return
		}
		err = config.verifyToken(rawIDToken)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Failed to verify ID Token: %s", err.Error())
		}

		cookie := http.Cookie{
			Name:  cookieName,
			Path:  "/",
			Value: rawIDToken,
		}
		http.SetCookie(w, &cookie)
		if r.URL.Query().Get("rd") != "" {
			http.Redirect(w, r, r.URL.Query().Get("rd"), http.StatusFound)
			return
		}
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	w.WriteHeader(http.StatusForbidden)
	fmt.Fprint(w, http.StatusText(http.StatusForbidden))
}
