package main

import (
	"flag"
	"github.com/vivowares/octopus/configs"
	"github.com/vivowares/octopus/connections"
	"github.com/vivowares/octopus/handlers"
	"github.com/vivowares/octopus/models"
	. "github.com/vivowares/octopus/utils"
	"github.com/zenazn/goji/bind"
	"github.com/zenazn/goji/graceful"
	"github.com/zenazn/goji/web"
	"github.com/zenazn/goji/web/middleware"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"runtime"
	"strconv"
	"time"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	configFile := flag.String("conf", "", "config file location")
	flag.Parse()
	if len(*configFile) > 0 {
		PanicIfErr(configs.InitializeConfig(*configFile))
	} else {
		pwd, err := os.Getwd()
		PanicIfErr(err)
		*configFile = path.Join(pwd, "configs", "octopus_development.yml")
		PanicIfErr(configs.InitializeConfig(*configFile))
	}

	PanicIfErr(models.InitializeDB())
	PanicIfErr(models.InitializeIndexDB())
	PanicIfErr(connections.InitializeCM())

	go func() {
		log.Printf("Octopus started listenning to port %s", configs.Config.Service.Host)

		graceful.Serve(
			bind.Socket(":"+strconv.Itoa(configs.Config.Service.HttpPort)),
			HttpRouter(),
		)
	}()

	go func() {
		log.Printf("Connection Manager started listenning to port %d", configs.Config.Service.WsPort)
		// http.ListenAndServe(":"+strconv.Itoa(configs.Config.Service.WsPort), WsRouter())
		graceful.Serve(
			bind.Socket(":"+strconv.Itoa(configs.Config.Service.WsPort)),
			WsRouter(),
		)
	}()

	graceful.HandleSignals()
	graceful.PreHook(func() { log.Printf("Octopus received signal, gracefully stopping...") })

	graceful.PostHook(func() {
		connections.CM.Close()
		log.Printf("Waiting for websockets to drain...")
		time.Sleep(3 * time.Second)
		log.Printf("Connection Manager closed.")
	})
	graceful.PostHook(func() { models.CloseDB() })
	graceful.PostHook(func() { models.CloseIndexDB() })
	graceful.PostHook(func() { log.Printf("Octopus stopped") })
	graceful.PostHook(func() { removePidFile() })

	createPidFile()

	graceful.Wait()
}

func WsRouter() http.Handler {
	wsRouter := web.New()
	wsRouter.Use(middleware.RequestID)
	wsRouter.Use(middleware.Logger)
	wsRouter.Use(middleware.Recoverer)
	wsRouter.Use(middleware.AutomaticOptions)
	wsRouter.Get("/heartbeat", handlers.HeartBeatWs)
	// wsRouter.Get("/ws/:channel_id/:device_id", handlers.WsHandler)

	wsRouter.Compile()

	return wsRouter
}

func HttpRouter() http.Handler {
	httpRouter := web.New()
	httpRouter.Use(middleware.RequestID)
	httpRouter.Use(middleware.Logger)
	httpRouter.Use(middleware.Recoverer)
	httpRouter.Use(middleware.AutomaticOptions)
	httpRouter.Get("/heartbeat", handlers.HeartBeatHttp)

	httpRouter.Get("/channels", handlers.ListChannels)
	httpRouter.Post("/channels", handlers.CreateChannel)
	httpRouter.Get("/channels/:id", handlers.GetChannel)
	httpRouter.Delete("/channels/:id", handlers.DeleteChannel)
	httpRouter.Put("/channels/:id", handlers.UpdateChannel)

	httpRouter.Compile()

	return httpRouter
}

func createPidFile() error {
	pid := os.Getpid()
	return ioutil.WriteFile(configs.Config.Service.PidFile, []byte(strconv.Itoa(pid)), 0644)
}

func removePidFile() error {
	return os.Remove(configs.Config.Service.PidFile)
}
