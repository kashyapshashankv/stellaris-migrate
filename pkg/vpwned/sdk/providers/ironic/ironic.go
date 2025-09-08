package ironic

import (
	"github.com/kashyapshashankv/stellaris-migrate/pkg/vpwned/sdk/providers"
	"github.com/kashyapshashankv/stellaris-migrate/pkg/vpwned/sdk/providers/base"
)

type IronicProvider struct {
	base.UnimplementedBaseProvider
}

func init() {
	providers.RegisterProvider("ironic", &IronicProvider{})
}
