package provider

// This file is mostly taken from the official tfe provider, licensed MPL2.

import (
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"github.com/hashicorp/go-tfe"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/logging"
	"github.com/hashicorp/terraform-svchost"
	"github.com/hashicorp/terraform-svchost/auth"
	"github.com/hashicorp/terraform-svchost/disco"
)

const (
	defaultHostname      = "app.terraform.io"
	defaultSSLSkipVerify = false
)

var (
	tfeServiceIDs       = []string{"tfe.v2.2"}
	errMissingAuthToken = errors.New("Required token could not be found. Please set the token using an input variable in the provider configuration block or by using the TFE_TOKEN environment variable.")
)

func getClient(version string, tfeHost, token string, insecure bool) (*tfe.Client, error) {
	h := tfeHost
	if tfeHost == "" {
		if os.Getenv("TFE_HOSTNAME") != "" {
			h = os.Getenv("TFE_HOSTNAME")
		} else {
			h = defaultHostname
		}
	}

	log.Printf("[DEBUG] Configuring client for host %q", h)

	// Parse the hostname for comparison,
	hostname, err := svchost.ForComparison(h)
	if err != nil {
		return nil, err
	}

	providerUaString := fmt.Sprintf(
		"terraform-provider-multispace/%s",
		version,
	)

	httpClient := tfe.DefaultConfig().HTTPClient
	transport := httpClient.Transport.(*http.Transport)
	if transport.TLSClientConfig == nil {
		transport.TLSClientConfig = &tls.Config{}
	}

	// If ssl_skip_verify is false, it is either set that way in configuration
	// or unset. Check the environment to see if it was set to true there.
	// Strictly speaking, this means that the env var can override an explicit
	// 'false' in configuration (which is not true of the other settings), but
	// that's how it goes with a boolean zero value.
	if !insecure && os.Getenv("TFE_SSL_SKIP_VERIFY") != "" {
		v := os.Getenv("TFE_SSL_SKIP_VERIFY")
		insecure, err = strconv.ParseBool(v)
		if err != nil {
			return nil, err
		}
	}
	if insecure {
		log.Printf("[DEBUG] Warning: Client configured to skip certificate verifications")
	}
	transport.TLSClientConfig.InsecureSkipVerify = insecure

	// Get the Terraform CLI configuration.
	config := cliConfig()

	// Create a new credential source and service discovery object.
	credsSrc := credentialsSource(config)
	services := disco.NewWithCredentialsSource(credsSrc)
	services.SetUserAgent(providerUaString)
	services.Transport = logging.NewTransport("TFE Discovery", transport)

	// Add any static host configurations service discovery object.
	for userHost, hostConfig := range config.Hosts {
		host, err := svchost.ForComparison(userHost)
		if err != nil {
			// ignore invalid hostnames.
			continue
		}
		services.ForceHostServices(host, hostConfig.Services)
	}

	// Discover the Terraform Enterprise address.
	host, err := services.Discover(hostname)
	if err != nil {
		return nil, err
	}

	// Get the full Terraform Enterprise service address.
	var address *url.URL
	var discoErr error
	for _, tfeServiceID := range tfeServiceIDs {
		service, err := host.ServiceURL(tfeServiceID)
		if _, ok := err.(*disco.ErrVersionNotSupported); !ok && err != nil {
			return nil, err
		}
		// If discoErr is nil we save the first error. When multiple services
		// are checked and we found one that didn't give an error we need to
		// reset the discoErr. So if err is nil, we assign it as well.
		if discoErr == nil || err == nil {
			discoErr = err
		}
		if service != nil {
			address = service
			break
		}
	}

	// When we don't have any constraints errors, also check for discovery
	// errors before we continue.
	if discoErr != nil {
		return nil, discoErr
	}

	// If a token wasn't set in the provider configuration block, try and fetch it
	// from the environment or from Terraform's CLI configuration or
	// configured credential helper.
	if token == "" {
		if os.Getenv("TFE_TOKEN") != "" {
			log.Printf("[DEBUG] TFE_TOKEN used for token value")
			token = os.Getenv("TFE_TOKEN")
		} else {
			log.Printf("[DEBUG] Attempting to fetch token from Terraform CLI configuration for hostname %s...", hostname)
			creds, err := services.CredentialsForHost(hostname)
			if err != nil {
				log.Printf("[DEBUG] Failed to get credentials for %s: %s (ignoring)", hostname, err)
			}
			if creds != nil {
				token = creds.Token()
			}
		}
	}

	// If we still don't have a token at this point, we return an error.
	if token == "" {
		return nil, errMissingAuthToken
	}

	// Wrap the configured transport to enable logging.
	httpClient.Transport = logging.NewTransport("TFE", transport)

	// Create a new TFE client config
	cfg := &tfe.Config{
		Address:    address.String(),
		Token:      token,
		HTTPClient: httpClient,
	}

	// Create a new TFE client.
	client, err := tfe.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	client.RetryServerErrors(true)
	return client, nil
}

func credentialsSource(config *Config) auth.CredentialsSource {
	creds := auth.NoCredentials

	// Add all configured credentials to the credentials source.
	if len(config.Credentials) > 0 {
		staticTable := map[svchost.Hostname]map[string]interface{}{}
		for userHost, creds := range config.Credentials {
			host, err := svchost.ForComparison(userHost)
			if err != nil {
				// We expect the config was already validated by the time we get
				// here, so we'll just ignore invalid hostnames.
				continue
			}
			staticTable[host] = creds
		}
		creds = auth.StaticCredentialsSource(staticTable)
	}

	return creds
}
