/*
Package config handles API server configuration binding and loading.

The config can be read from an environment (twelve-factor methodology),
from a command flags, and from a config file. We strongly recommend

*/
package config

import (
	"flag"
	"github.com/spf13/viper"
	"log"
	"time"
)

// default configuration elements and keys
const (
	appName        = "FantomAPI"
	configFileName = "apiserver"

	// configuration options
	keyConfigFilePath    = "cfg"
	keyBindAddress       = "server.bind"
	keyDomainAddress     = "server.domain"
	keyApiStateOrigin    = "server.origin"
	keyLoggingLevel      = "log.level"
	keyLoggingFormat     = "log.format"
	keyLachesisUrl       = "lachesis.url"
	keyMongoUrl          = "mongo.url"
	keyMongoDatabase     = "mongo.db"
	keyCorsAllowOrigins  = "cors.origins"
	keyCacheEvictionTime = "cache.eviction"
	keySolCompilerPath   = "sol.compiler"
	keyVotingSources     = "voting.sources"

	// contract validation details
	keyContractValidatorRepository  = "contracts.repository"
	keyContractValidatorSigKey      = "contracts.validation.key"
	keyContractValidatorSigPassword = "contracts.validation.password"
)

// Config defines configuration options structure for Fantom API server.
type Config struct {
	// AppName holds the name of the application
	AppName string

	// BindAddress holds the API server network binding address
	BindAddress string

	// DomainName holds the domain of the API server deployment.
	DomainName string

	// LoggingLevel and LoggingFormat hold configuration for the API server logger
	LoggingLevel  string
	LoggingFormat string

	// LachesisUrl holds address of the Lachesis node we want to communicate with
	LachesisUrl string

	// MongoUrl holds address of the MongoDB we use for persistent storage
	MongoUrl string

	// MongoDatabase represents the name of the database used for the API persistent data storage.
	MongoDatabase string

	// CorsAllowOrigins keeps list of origins allowed to make requests on the server HTTP interface
	CorsAllowOrigins []string

	// CacheEvictionTime specifies the time after which entry can be evicted from in-memory cache
	CacheEvictionTime time.Duration

	// SolCompilerPath represents the path to sol compiler for smart contract validation.
	SolCompilerPath string

	// VotingSources represents a list of addresses used to deploy voting smart contracts
	// for official Fantom ballots.
	VotingSources []string

	// ApiStateOrigin represents request origin used on state syncing events.
	ApiStateOrigin string

	// ContractValidatorRepository is the address of the on-chain contract validation repo.
	ContractValidatorRepository string

	// ContractValidatorSigKey is the path to the contract validation signing key file.
	ContractValidatorSigKey string

	// ContractValidatorSigPassword is the password for the contract validation signing key.
	ContractValidatorSigPassword string
}

// Load provides a loaded configuration for Fantom API server.
func Load() (*Config, error) {
	// Get the config reader
	cfg := reader()

	// set default values
	applyDefaults(cfg)

	// Try to read the file
	if err := cfg.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; ignore the error, we may not need the config file
			log.Print("configuration file not found, using default values")
		} else {
			// Config file was found but another error was produced
			log.Printf("can not read the server configuration")
			return nil, err
		}
	}

	// Build and return the config structure
	return &Config{
		AppName:           appName,
		BindAddress:       cfg.GetString(keyBindAddress),
		DomainName:        cfg.GetString(keyDomainAddress),
		LoggingLevel:      cfg.GetString(keyLoggingLevel),
		LoggingFormat:     cfg.GetString(keyLoggingFormat),
		LachesisUrl:       cfg.GetString(keyLachesisUrl),
		MongoUrl:          cfg.GetString(keyMongoUrl),
		MongoDatabase:     cfg.GetString(keyMongoDatabase),
		CorsAllowOrigins:  cfg.GetStringSlice(keyCorsAllowOrigins),
		CacheEvictionTime: cfg.GetDuration(keyCacheEvictionTime),
		SolCompilerPath:   cfg.GetString(keySolCompilerPath),
		ApiStateOrigin:    cfg.GetString(keyApiStateOrigin),
		VotingSources:     cfg.GetStringSlice(keyVotingSources),

		// contract validation related configuration
		ContractValidatorRepository:  cfg.GetString(keyContractValidatorRepository),
		ContractValidatorSigKey:      cfg.GetString(keyContractValidatorSigKey),
		ContractValidatorSigPassword: cfg.GetString(keyContractValidatorSigPassword),
	}, nil
}

// reader provides instance of the config reader.
// It accepts an explicit path to a config file if it was requested by `cfg` flag.
func reader() *viper.Viper {
	// make new Viper
	cfg := viper.New()

	// what is the expected name of the config file
	cfg.SetConfigName(configFileName)

	// where to look for common files
	cfg.AddConfigPath(defaultConfigDir())
	cfg.AddConfigPath(".")

	// Try to get an explicit configuration file path if present
	var cfgPath string
	flag.StringVar(&cfgPath, keyConfigFilePath, "", "Path to a configuration file")
	flag.Parse()

	// Any path found?
	cfg.SetConfigFile(cfgPath)

	return cfg
}
