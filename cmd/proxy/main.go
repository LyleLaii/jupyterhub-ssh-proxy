package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"

	"jupyterhub-ssh-proxy/jupyterhubserver"
	"jupyterhub-ssh-proxy/sshproxy"

	zaplogger "github.com/lylelaii/golang_utils/logger/v1/zaplogger"
	version "github.com/lylelaii/golang_utils/version/v1"
	"github.com/spf13/viper"
	"golang.org/x/crypto/ssh"
	"gopkg.in/alecthomas/kingpin.v2"
)

const SERVERNAME = "JupyterHub-SSH-Proxy"

func main() {
	os.Exit(run())
}

func run() int {
	var (
		listen        = kingpin.Flag("listen", "listen address. Default is :8080").Default(":8080").String()
		cfg           = kingpin.Flag("config.file", "JupyterHub-ssh-proxy configuration file path. Default is ./etc/config.yaml").Default("./etc/config.yaml").String()
		runMode       = kingpin.Flag(zaplogger.RunModeFlagName, zaplogger.RunModeFlagHelp).Default("release").String()
		logLevel      = kingpin.Flag(zaplogger.LevelFlagName, zaplogger.LevelFlagHelp).Default("Info").String()
		logMaxBackups = kingpin.Flag(zaplogger.LogMaxBackupsFlagName, zaplogger.LogMaxBackupsFlagHelp).Default("5").Int()
		logMaxDays    = kingpin.Flag(zaplogger.LogMaxDaysFlagName, zaplogger.LogMaxDaysFlagHelp).Default("30").Int()
	)

	kingpin.Version(version.Print())
	kingpin.CommandLine.GetFlag("help").Short('h')
	kingpin.Parse()

	viper.New()
	viper.SetConfigFile(*cfg)
	viper.SetConfigType("yaml")
	if err := viper.ReadInConfig(); err != nil {
		panic("Failed to load config")
	}

	loggerConfig := zaplogger.ConfigZap(SERVERNAME, zaplogger.NewRunConf(*logLevel, *runMode, *logMaxBackups, *logMaxDays))
	logger := zaplogger.NewZapSugarLogger(loggerConfig)

	var jhConfig jupyterhubserver.JupyterHubServerConfig
	if err := viper.UnmarshalKey("jupyterhub", &jhConfig); err != nil {
		logger.Error(SERVERNAME, fmt.Sprintf("Error load jupyterhub config: %s", err))
	}

	jhServer := jupyterhubserver.NewJupyterHubServer(jhConfig, logger)

	privateBytes, err := ioutil.ReadFile(viper.GetString("host_key_path"))
	if err != nil {
		panic("Failed to load private key")
	}

	private, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		panic("Failed to parse private key")
	}

	srv := sshproxy.NewSshProxyServer(*listen, private, jhServer, logger)

	srvc := make(chan struct{})

	go func() {
		logger.Info(SERVERNAME, fmt.Sprintf("Server start,listening on: %s", *listen))

		if err := srv.ListenAndServe(); err != nil {
			logger.Error(SERVERNAME, fmt.Sprintf("Start Error: %s", err.Error()))
			close(srvc)
		}

		defer func() {
			if err := srv.Close(); err != nil {
				logger.Error(SERVERNAME, fmt.Sprintf("Error when closing server: %+v", err))
			}
		}()

	}()

	var (
		hup      = make(chan os.Signal, 1)
		hupReady = make(chan bool)
		term     = make(chan os.Signal, 1)
	)
	signal.Notify(hup, syscall.SIGHUP)
	signal.Notify(term, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-hupReady
		for {
			<-hup
			logger.Info(SERVERNAME, "receive hup signal")
			// select {
			// case <-hup:
			// 	// TODO
			// 	// ignore error, already logged in `reload()`
			// 	logger.Info(SERVERNAME, "receive hup signal")
			// }
		}
	}()

	// Wait for reload or termination signals.
	close(hupReady) // Unblock SIGHUP handler.

	for {
		select {
		case <-term:
			logger.Info(SERVERNAME, "Received SIGTERM, exiting gracefully...")
			return 0
		case <-srvc:
			return 1
		}
	}

}
