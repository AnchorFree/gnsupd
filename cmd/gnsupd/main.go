package main

import (
	"context"
	"encoding/json"
	"github.com/anchorfree/golang/pkg/jsonlog"
	"github.com/kelseyhightower/envconfig"
	"github.com/projectcalico/libcalico-go/lib/apis/v3"
	client "github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/libcalico-go/lib/options"
	"io/ioutil"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

type Config struct {
	ConfigDir  string `default:"/etc/ipsets" split_words:"true"`
	ExtraLabel string `default:"" split_words:"true"`
}

type App struct {
	config Config
	log    jsonlog.Logger
}

// NewApp initializes the logger and parses the configuration.
func NewApp() (*App, error) {

	log := &jsonlog.StdLogger{}
	log.Init("gnsupd", false, false, nil)

	app := &App{config: Config{}}
	err := envconfig.Process("gnsupd", &app.config)
	if err != nil {
		log.Fatal("failed to initialize", err)
	}
	app.log = log
	return app, err

}

// ScanSetsDir returns a list of all files that have '.json' suffix in a given
// directory. The names in the returned list are trimmed of the suffix.
func ScanSetsDir(dir string) ([]string, error) {

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	names := []string{}
	for _, f := range files {
		name := f.Name()
		if strings.HasSuffix(name, ".json") {
			names = append(names, name[:len(name)-5])
		}
	}
	return names, nil

}

// CreateGNSFromFile returns an initialized v3.GlobalNetworkSet structure where
// networks are defined from the provided file. The file must contain "nets" JSON
// array with networks, e.g.: { "nets" : [ "10.100.11.0/24", "192.168.7.0/24" ] }
func CreateGNSFromFile(filename, setName, extraLabel string) (*v3.GlobalNetworkSet, error) {

	rawJSON, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	GNS := v3.NewGlobalNetworkSet()
	GNS.ObjectMeta.Name = setName
	GNS.ObjectMeta.Labels = map[string]string{setName: "true"}
	if extraLabel != "" {
		GNS.ObjectMeta.Labels[extraLabel] = "true"
	}
	HostList := v3.GlobalNetworkSetSpec{}
	err = json.Unmarshal(rawJSON, &HostList)
	if err != nil {
		return nil, err
	}
	GNS.Spec = HostList
	return GNS, nil

}

// UpdateGNS actually updates (or creates) provided GNS resource via
// making a request to the kube API.
func UpdateGNS(setName string, GNS *v3.GlobalNetworkSet) error {

	cl, err := client.NewFromEnv()
	if err != nil {
		return err
	}

	GNSInterface := cl.GlobalNetworkSets()
	ctx := context.TODO()
	existingGNS, err := GNSInterface.Get(ctx, setName, options.GetOptions{})
	if err != nil {
		_, err = GNSInterface.Create(ctx, GNS, options.SetOptions{})
	} else {
		// When we update an already existing resource almost
		// all of the fields of GNS.ObjectMeta should be set,
		// and the easiest way to do that is just to redefine the networks
		// in the resource we got from the Get request.
		existingGNS.Spec = GNS.Spec
		_, err = GNSInterface.Update(ctx, existingGNS, options.SetOptions{})
	}
	return err

}

// UpdateAllSets runs UpdateGNS for every file found in the
// app.config.ConfigDir directory.
func UpdateAllSets(app *App) {

	sets, err := ScanSetsDir(app.config.ConfigDir)
	if err == nil {
		for _, set := range sets {
			GNS, err := CreateGNSFromFile(app.config.ConfigDir+"/"+set+".json", set, app.config.ExtraLabel)
			if err == nil {
				err = UpdateGNS(set, GNS)
				if err != nil {
					app.log.Error("failed to update GNS "+set+":", err)
				} else {
					app.log.Info("updated GNS " + set)
				}
			} else {
				app.log.Error("failed to create GNS from file "+app.config.ConfigDir+"/"+set+".json", err)
			}
		}
	} else {
		app.log.Error("error scanning GNSUPD_CONFIG_DIR:", err)
	}

}

func main() {

	app, err := NewApp()
	if err != nil {
		app.log.Fatal("can't initialize application", err)
	}

	sigHUP := make(chan os.Signal, 1)
	manualTrigger := make(chan bool, 1)
	quit := make(chan bool, 1)

	updateOnRequest := func(a *App) {
		for {
			select {
			case <-sigHUP:
				UpdateAllSets(a)
			case <-manualTrigger:
				UpdateAllSets(a)
			}
		}
	}
	signal.Notify(sigHUP, syscall.SIGHUP)

	go updateOnRequest(app)
	manualTrigger <- true
	_ = <-quit

}
