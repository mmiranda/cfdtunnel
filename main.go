package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"time"

	log "github.com/sirupsen/logrus"
	"gopkg.in/ini.v1"
)

const (
	IniConfigFile = ".cfdtunnel/config"
	DefaultPort   = "5555"
)

var (
	LogLevel   = log.WarnLevel
	AppVersion = "Development"
)

type tunnelConfig struct {
	host string
	port string
}

type config struct {
	ini *ini.File
}

type Arguments struct {
	profile *string
	command string
	args    []string
}

func init() {
	log.SetOutput(os.Stdout)
	log.SetLevel(LogLevel)
}

func main() {
	args := flagArguments()
	log.SetLevel(LogLevel)

	config, err := readIniConfigFile(getHomePathIniFile(IniConfigFile))

	if err != nil {
		log.Fatalf("An error occured reading your INI file: %v", err.Error())
	}

	tunnelConfig, err := config.readConfigSection(*args.profile)

	if err != nil {
		log.Fatalf("An error occured reading your INI file: %v", err.Error())
	}

	cmd := startProxyTunnel(tunnelConfig)
	os.Setenv("HTTPS_PROXY", "socks5://127.0.0.1:"+tunnelConfig.port)
	runSubCommand(args)

	// Kill it:
	commandKill(cmd)
}

func commandKill(cmd *exec.Cmd) {
	log.Debugf("Trying to kill PID: %v", cmd.Process.Pid)

	if err := cmd.Process.Kill(); err != nil {
		log.Fatal("failed to kill process: ", err)
	}
}

func runSubCommand(args Arguments) {
	log.Debugf("Running subcommand: %v", args.command)
	if !checkSubCommandExists(args.command) {
		os.Exit(1)
	}

	output, err := exec.Command(args.command, args.args...).CombinedOutput()

	fmt.Println(string(output))

	if err != nil {
		log.Fatalf("An error occurred trying to run the command %v: %v", args.command, err)
	}

}

// readIniConfigFile checks if the file exists
func readIniConfigFile(configFile string) (config, error) {

	cfg, err := ini.Load(configFile)

	if err != nil {
		return config{}, err
	}

	return config{
		ini: cfg,
	}, err
}

func startProxyTunnel(tunnelConfig tunnelConfig) *exec.Cmd {
	log.Debugf("Starting proxy tunnel for %v on port: %v", tunnelConfig.host, tunnelConfig.port)

	cmd := exec.Command("cloudflared", "access", "tcp", "--hostname", tunnelConfig.host, "--url", "127.0.0.1:"+tunnelConfig.port)

	err := cmd.Start()

	// Hacky thing to wait for the first process start correctly
	time.Sleep(1 * time.Second)

	if err != nil {
		log.Fatalf("Could not start cloudflared: %v", err.Error())
	}

	log.Debugf("cloudflared process running on PID: %v", cmd.Process.Pid)

	return cmd
}

func (cfg config) readConfigSection(section string) (tunnelConfig, error) {

	secs, err := cfg.ini.GetSection(section)

	if err != nil {
		log.Debugf("An error occured: %v", err.Error())
		return tunnelConfig{}, err
	}

	host, _ := secs.GetKey("host")

	port := secs.Key("port").Validate(func(port string) string {
		if len(port) == 0 {
			return DefaultPort
		}
		return port
	})

	return tunnelConfig{
		host: host.String(),
		port: port,
	}, nil
}

func getHomePathIniFile(file string) string {
	home, _ := os.UserHomeDir()

	return home + "/" + file
}

func flagArguments() Arguments {
	profile := flag.String("profile", "", "Which cfdtunnel profile to use")
	version := flag.Bool("version", false, "Show cfdtunnel version")
	debug := flag.Bool("debug", false, "Enable Debug mode")

	flag.Parse()

	if *version {
		fmt.Println(AppVersion)
		os.Exit(0)
	}

	if *debug {
		LogLevel = log.DebugLevel
	}

	if *profile == "" {
		fmt.Println("Usage: cfdtunnel --profile xxx command args")
		os.Exit(1)
		return Arguments{}
	}

	args := flag.Args()

	return Arguments{
		profile: profile,
		command: args[0],
		args:    args[1:],
	}
}

func checkSubCommandExists(command string) bool {
	_, err := exec.LookPath(command)
	if err != nil {
		log.Errorf("An error occured: %v", err.Error())
		return false
	}

	return true
}
