package metrics

import (
	"flag"

	"github.com/safing/portbase/config"
)

// Configuration Keys
var (
	CfgOptionInstanceKey   = "core/metrics/instance"
	instanceOption         config.StringOption
	cfgOptionInstanceOrder = 0

	CfgOptionPushKey   = "core/metrics/push"
	pushOption         config.StringOption
	cfgOptionPushOrder = 0

	pushFlag     string
	instanceFlag string
)

func init() {
	flag.StringVar(&pushFlag, "push-metrics", "", "Set default URL to push prometheus metrics to.")
	flag.StringVar(&instanceFlag, "metrics-instance", "", "Set the default global instance label.")
}

func prepConfig() error {
	err := config.Register(&config.Option{
		Name:            "Metrics Instance Name",
		Key:             CfgOptionInstanceKey,
		Description:     "Define the prometheus instance label for exported metrics. Please note that changing the instance name will reset persisted metrics.",
		OptType:         config.OptTypeString,
		ExpertiseLevel:  config.ExpertiseLevelExpert,
		ReleaseLevel:    config.ReleaseLevelStable,
		DefaultValue:    instanceFlag,
		RequiresRestart: true,
		Annotations: config.Annotations{
			config.DisplayOrderAnnotation: cfgOptionInstanceOrder,
			config.CategoryAnnotation:     "Metrics",
		},
		ValidationRegex: "^(" + prometheusBaseFormt + ")?$",
	})
	if err != nil {
		return err
	}
	instanceOption = config.Concurrent.GetAsString(CfgOptionInstanceKey, instanceFlag)

	err = config.Register(&config.Option{
		Name:            "Push Metrics",
		Key:             CfgOptionPushKey,
		Description:     "Push metrics to this URL in the prometheus format.",
		OptType:         config.OptTypeString,
		ExpertiseLevel:  config.ExpertiseLevelExpert,
		ReleaseLevel:    config.ReleaseLevelStable,
		DefaultValue:    pushFlag,
		RequiresRestart: true,
		Annotations: config.Annotations{
			config.DisplayOrderAnnotation: cfgOptionPushOrder,
			config.CategoryAnnotation:     "Metrics",
		},
	})
	if err != nil {
		return err
	}
	pushOption = config.Concurrent.GetAsString(CfgOptionPushKey, pushFlag)

	return nil
}
