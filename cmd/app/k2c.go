package app 

import (
	"github.com/yangzhares/kube2consul/cmd/app/options"
)

/*
// NewKube2ConsulCommand creates cobra command
func NewKube2ConsulCommand() *cobra.Command {
	app := options.NewAPP()
	app.AddFlags(pflag.CommandLine)
	cmd := &cobra.Command{
		Use: "kube2consul",
		Long: `Kube2Consul monitors Kubernetes Master to register headless service into Consul to do service discovery.`,
		Run: func(cmd *cobra.Command, args []string) {
		},
	}
	return cmd
}
*/

// Start kube2consul
func Start(app *options.APP) {
    app.Start()
}