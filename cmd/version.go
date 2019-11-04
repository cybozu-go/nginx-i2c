package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

var (
	// CurrentVersion is a build-time string representing the current version
	CurrentVersion string
	versionCmd = &cobra.Command{
		Use: "version",
		Short: "display the version number",
		Args: cobra.NoArgs,
		Run: showVersion,
	}
)


func showVersion(cmd *cobra.Command, args[] string) {
	fmt.Printf("nginx-i2c version %s\n", CurrentVersion)
}