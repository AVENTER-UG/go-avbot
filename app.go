package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"

	_ "go-avbot/services/echo"
	_ "go-avbot/services/gitea"
	_ "go-avbot/services/ollama"
	_ "go-avbot/services/pentest"
	_ "go-avbot/services/unifi_protect"
	_ "go-avbot/services/wekan"
	"go-avbot/types"

	"go-avbot/api"
	"go-avbot/api/handlers"
	"go-avbot/clients"
	"go-avbot/database"
	"go-avbot/polling"

	"github.com/AVENTER-UG/util/util"
	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

// BindAddress is the Bind Address of the bot
var BindAddress string

// DatabaseType is by default sqlite3
var DatabaseType string

// DatabaseURL is the url of the database :-)
var DatabaseURL string

// BaseURL is the url format of the database query
var BaseURL string

// ConfigFile is the bots config file in yaml format
var ConfigFile string

// MinVersion is the BuildVersion Number
var MinVersion string

// loadFromConfig loads a config file and returns a ConfigFile
func loadFromConfig(db *database.ServiceDB, configFilePath string) (*api.ConfigFile, error) {
	// ::Horrible hacks ahead::
	// The config is represented as YAML, and we want to convert that into NEB types.
	// However, NEB types make liberal use of json.RawMessage which the YAML parser
	// doesn't like. We can't implement MarshalYAML/UnmarshalYAML as a custom type easily
	// because YAML is insane and supports numbers as keys. The YAML parser therefore has the
	// generic form of map[interface{}]interface{} - but the JSON parser doesn't know
	// how to parse that.
	//
	// The hack that follows gets around this by type asserting all parsed YAML keys as
	// strings then re-encoding/decoding as JSON. That is:
	// YAML bytes -> map[interface]interface -> map[string]interface -> JSON bytes -> NEB types

	// Convert to YAML bytes
	contents, err := os.ReadFile(configFilePath)
	if err != nil {
		return nil, err
	}

	// Convert to map[interface]interface
	var cfg map[interface{}]interface{}
	if err = yaml.Unmarshal(contents, &cfg); err != nil {
		return nil, fmt.Errorf("Failed to unmarshal YAML: %s", err)
	}

	// Convert to map[string]interface
	dict := convertKeysToStrings(cfg)

	// Convert to JSON bytes
	b, err := json.Marshal(dict)
	if err != nil {
		return nil, fmt.Errorf("Failed to marshal config as JSON: %s", err)
	}

	// Finally, Convert to NEB types
	var c api.ConfigFile
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("Failed to convert to config file: %s", err)
	}

	// sanity check (at least 1 client and 1 service)
	if len(c.Clients) == 0 || len(c.Services) == 0 {
		return nil, fmt.Errorf("At least 1 client and 1 service must be specified")
	}

	return &c, nil
}

func convertKeysToStrings(iface interface{}) interface{} {
	obj, isObj := iface.(map[interface{}]interface{})
	if isObj {
		strObj := make(map[string]interface{})
		for k, v := range obj {
			strObj[k.(string)] = convertKeysToStrings(v) // handle nested objects
		}
		return strObj
	}

	arr, isArr := iface.([]interface{})
	if isArr {
		for i := range arr {
			arr[i] = convertKeysToStrings(arr[i]) // handle nested objects
		}
		return arr
	}
	return iface // base type like string or number
}

func insertServicesFromConfig(clis *clients.Clients, serviceReqs []api.ConfigureServiceRequest) error {
	for i, s := range serviceReqs {
		if err := s.Check(); err != nil {
			return fmt.Errorf("config: Service[%d] : %s", i, err)
		}
		service, err := types.CreateService(s.ID, s.Type, s.UserID, s.Config)
		if err != nil {
			return fmt.Errorf("config: Service[%d] : %s", i, err)
		}

		// Fetch the client for this service and register/poll
		c, err := clis.Client(s.UserID)
		if err != nil {
			return fmt.Errorf("config: Service[%d] : %s", i, err)
		}

		if err = service.Register(nil, c); err != nil {
			return fmt.Errorf("config: Service[%d] : %s", i, err)
		}
		if _, err := database.GetServiceDB().StoreService(service); err != nil {
			return fmt.Errorf("config: Service[%d] : %s", i, err)
		}
		service.PostRegister(nil)
	}
	return nil
}

func loadDatabase(databaseType, databaseURL, configYAML string) (*database.ServiceDB, error) {
	if configYAML != "" {
		databaseType = "sqlite3"
		databaseURL = ":memory:?_busy_timeout=5000"
	}

	db, err := database.Open(databaseType, databaseURL)
	if err == nil {
		database.SetServiceDB(db) // set singleton
	}
	return db, err
}

func setup(mux *http.ServeMux, matrixClient *http.Client) {
	err := types.BaseURL(BaseURL)
	if err != nil {
		log.WithError(err).Panic("Failed to get base url")
	}

	db, err := loadDatabase(DatabaseType, DatabaseURL, ConfigFile)
	if err != nil {
		log.WithError(err).Panic("Failed to open database")
	}

	// Populate the database from the config file if one was supplied.
	var cfg *api.ConfigFile
	if ConfigFile != "" {
		if cfg, err = loadFromConfig(db, ConfigFile); err != nil {
			log.WithError(err).WithField("config_file", ConfigFile).Panic("Failed to load config file")
		}
		if err := db.InsertFromConfig(cfg); err != nil {
			log.WithError(err).Panic("Failed to persist config data into in-memory DB")
		}
		log.Info("Inserted ", len(cfg.Clients), " clients")
		log.Info("Inserted ", len(cfg.Realms), " realms")
		log.Info("Inserted ", len(cfg.Sessions), " sessions")
	}

	clients := clients.New(db, matrixClient)
	if err := clients.Start(); err != nil {
		log.WithError(err).Panic("Failed to start up clients")
	}

	// Read exclusively from the config file if one was supplied.
	// Otherwise, add HTTP listeners for new Services/Sessions/Clients/etc.
	if ConfigFile != "" {
		if err := insertServicesFromConfig(clients, cfg.Services); err != nil {
			log.WithError(err).Panic("Failed to insert services")
		}

		log.Info("Inserted ", len(cfg.Services), " services")
	}
	polling.SetClients(clients)
	if err := polling.Start(); err != nil {
		log.WithError(err).Panic("Failed to start polling")
	}
	mux.HandleFunc("/test", util.Protect(handlers.Heartbeat))
	wh := handlers.NewWebhook(db, clients)
	mux.HandleFunc("/services/hooks/", util.Protect(wh.Handle))
}

func main() {

	BindAddress = util.Getenv("BIND_ADDRESS", "0.0.0.0:4050")
	BaseURL = util.Getenv("BASE_URL", "http://localhost:4050/")

	log.Infof("GO-AVBOT build %s (%s %s %s %s %s)", MinVersion, BindAddress, BaseURL, DatabaseType, DatabaseURL, ConfigFile)

	setup(http.DefaultServeMux, http.DefaultClient)
	log.Fatal(http.ListenAndServe(BindAddress, nil))
}
