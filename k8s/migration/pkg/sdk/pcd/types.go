// Package du provides utilities and types for interacting with Openstack Cloud services.
// It includes configuration types and helper functions for Openstack Cloud operations.
package du

// Info contains connection information for a Openstack Cloud instance,
// including the URL and whether to skip TLS certificate verification.
type Info struct {
	URL      string
	Insecure bool
}
