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
	"syscall"
)

type Config struct {
	NetworksFile string `required:"true" split_words:"true"`
	SetName      string `required:"true" split_words:"true"`
}

type App struct {
	config Config
	log    jsonlog.Logger
}

func NewApp() (*App, error) {

	log := &jsonlog.StdLogger{}
	log.Init("gnsupd", false, false, nil)

	app := &App{config: Config{}}
	err := envconfig.Process("gnsupd", &app.config)
	if err != nil {
		log.Fatal("failed to initialize", err)
	}
	return app, err

}

func CreateGNSFromFile(filename, setName string) (*v3.GlobalNetworkSet, error) {

	rawJSON, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	GNS := v3.NewGlobalNetworkSet()
	GNS.ObjectMeta.Name = setName
	GNS.ObjectMeta.Labels = map[string]string{setName: "true"}
	HostList := v3.GlobalNetworkSetSpec{}
	err = json.Unmarshal(rawJSON, &HostList)
	if err != nil {
		return nil, err
	}
	GNS.Spec = HostList
	return GNS, nil

}

func UpdateGNS(setName string, GNS *v3.GlobalNetworkSet) error {

	cl, err := client.NewFromEnv()
	if err != nil {
		return err
	}

	GNSInterface := cl.GlobalNetworkSets()
	ctx := context.TODO()
	_, err = GNSInterface.Get(ctx, setName, options.GetOptions{})
	if err != nil {
		_, err = GNSInterface.Create(ctx, GNS, options.SetOptions{})
	} else {
		_, err = GNSInterface.Update(ctx, GNS, options.SetOptions{})
	}
	return err

}

func main() {

	app, err := NewApp()
	if err != nil {
		app.log.Fatal("can't initialize application", err)
	}

	sigHUP := make(chan os.Signal, 1)
	manualTrigger := make(chan bool, 1)
	quit := make(chan bool, 1)

	updateGNS := func(app *App) {

		doUpdate := func(app *App) {

			GNS, err := CreateGNSFromFile(app.config.NetworksFile, app.config.SetName)
			if err == nil {
				err = UpdateGNS(app.config.SetName, GNS)
				if err != nil {
					app.log.Error("failed to update GNS", err)
				} else {
					app.log.Info("updated GNS")
				}
			} else {
				app.log.Error("failed to create GNS from file", err)
			}
		}

		for {
			select {
			case <-sigHUP:
				doUpdate(app)
			case <-manualTrigger:
				doUpdate(app)
			}
		}
	}
	signal.Notify(sigHUP, syscall.SIGHUP)

	go updateGNS(app)
	manualTrigger <- true
	_ = <-quit

}
