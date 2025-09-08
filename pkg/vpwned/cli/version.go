package cli

import (
	"context"
	"fmt"

	api "github.com/kashyapshashankv/stellaris-migrate/pkg/vpwned/api/proto/v1/service"
	"github.com/kashyapshashankv/stellaris-migrate/pkg/vpwned/server"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "show version",
	Long:  "show version",
	Run: func(cmd *cobra.Command, args []string) {
		v := server.VpwnedVersion{}
		ver, _ := v.Version(context.Background(), &api.VersionRequest{})
		fmt.Println("vpwctl version", ver.Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
