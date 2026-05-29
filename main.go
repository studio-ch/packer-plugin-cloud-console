package main

import (
	"fmt"
	"os"

	"github.com/hashicorp/packer-plugin-sdk/plugin"
	"github.com/hashicorp/packer-plugin-sdk/version"

	"github.com/studio-ch/packer-plugin-xcloud/builder"
)

var pluginVersion = version.NewPluginVersion("0.1.0", "", "")

func main() {
	pps := plugin.NewSet()
	pps.SetVersion(pluginVersion)
	pps.RegisterBuilder(plugin.DEFAULT_NAME, new(builder.Builder))
	if err := pps.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
