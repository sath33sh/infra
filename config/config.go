// This package provides an interface to read configuration stored in JSON
// format. It is designed as a light-weight wrapper around Viper library.
// For more details on Viper, see https://github.com/spf13/viper
//
// Note: This package panics during init if configuration file is not found.
//
package config

import (
	"fmt"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
)

// Configuration context.
type ConfigCtx struct {
	v *viper.Viper
}

// Base configuration context.
var Base ConfigCtx

func Read(path string) (*ConfigCtx, error) {
	ctx := &ConfigCtx{v: viper.New()}

	ctx.v.SetConfigType("json")
	ctx.v.SetConfigFile(path)
	err := ctx.v.ReadInConfig()
	if err != nil {
		return ctx, err
	}

	return ctx, nil
}

// Parse base configuration.
func parseBaseConfig(baseConfPath string) {
	if Base.v == nil {
		Base.v = viper.New()
	}

	Base.v.SetConfigType("json")
	Base.v.SetConfigFile(baseConfPath)
	err := Base.v.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("Failed to read base config: %s", err))
	}
}

// Initialize configuration.
func Init(baseConfPath string) {
	// Initialize base configuration.
	if baseConfPath != "" {
		parseBaseConfig(baseConfPath)
	}
}

func (cc *ConfigCtx) GetInt(module, key string, dflt int) int {
	if val := cc.v.GetStringMap(module)[key]; val != nil {
		return cast.ToInt(val)
	} else {
		return dflt
	}
}

func (cc *ConfigCtx) GetBool(module, key string, dflt bool) bool {
	if val := cc.v.GetStringMap(module)[key]; val != nil {
		return cast.ToBool(val)
	} else {
		return dflt
	}
}

func (cc *ConfigCtx) GetString(module, key string, dflt string) string {
	if val := cc.v.GetStringMapString(module)[key]; val != "" {
		return val
	} else {
		return dflt
	}
}

func (cc *ConfigCtx) GetStringSlice(module, key string, dflt []string) []string {
	if val := cc.v.GetStringMap(module)[key]; val != nil {
		return cast.ToStringSlice(val)
	} else {
		return dflt
	}
}

func (cc *ConfigCtx) UnmarshalKey(key string, data interface{}) error {
	return cc.v.UnmarshalKey(key, data)
}
