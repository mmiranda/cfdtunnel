package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"gopkg.in/ini.v1"
)

const (
	iniConfigFile          = ".cfdtunnel/config"
	localClientDefaultPort = "5555"
)

var (
	logLevel   = log.WarnLevel
	appVersion = "Development"
)

// TunnelConfig struct stores data to launch cloudflared process such as hostname and port.
// It also stores preset Environment Variables needed to use together with the tunnel consumer.
type TunnelConfig struct {
	host    string
	port    string
	envVars []string
}

type config struct {
	ini *ini.File
}

// Arguments struct stores the arguments passed to cfdtunel such as the profile to use, the command to run and the arguments for that command
type Arguments struct {
	profile *string
	command string
	args    []string
}

func init() {
	log.SetOutput(os.Stdout)
	log.SetLevel(logLevel)
}

func main() {
	args := flagArguments()
	log.SetLevel(logLevel)

	config, err := readIniConfigFile(getHomePathIniFile(iniConfigFile))

	if err != nil {
		log.Fatalf("An error occurred reading your INI file: %v", err.Error())
	}

	tunnelConfig, err := config.readConfigSection(*args.profile)

	if err != nil {
		log.Fatalf("An error occurred reading your INI file: %v", err.Error())
	}
	tunnelConfig.setupEnvironmentVariables()
	cmd := tunnelConfig.startProxyTunnel()
	args.runSubCommand()

	// Kill it:
	commandKill(cmd)
}

// commandKill Kills an specific *exec.Cmd command
func commandKill(cmd *exec.Cmd) {
	log.Debugf("Trying to kill PID: %v", cmd.Process.Pid)

	if err := cmd.Process.Kill(); err != nil {
		log.Fatal("failed to kill process: ", err)
	}
}

// runSubCommand Runs the SubCommand and its arguments passed to cfdtunnel
func (args Arguments) runSubCommand() {
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

// readIniConfigFile reads and load the config file.
func readIniConfigFile(configFile string) (config, error) {

	cfg, err := ini.ShadowLoad(configFile)

	if err != nil {
		return config{}, err
	}

	return config{
		ini: cfg,
	}, err
}

// setupEnvironmentVariables Sets every environment variables that are expected and informed on the config file
func (tunnelConfig TunnelConfig) setupEnvironmentVariables() {
	for _, env := range tunnelConfig.envVars {
		iniEnv := strings.Split(env, "=")
		log.Debugf("Exporting Environment variable: %v", env)
		os.Setenv(iniEnv[0], iniEnv[1])
	}
}

// startProxyTunnel Starts the proxy tunnel (cloudflared process) and return its command instance
func (tunnelConfig TunnelConfig) startProxyTunnel() *exec.Cmd {
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

// readConfigSection reads an specific section from a config file.
// It returns a tunnelConfig struct containing the hostname, port and any environment variable needed
func (cfg config) readConfigSection(section string) (TunnelConfig, error) {

	secs, err := cfg.ini.GetSection(section)

	if err != nil {
		log.Debugf("An error occurred: %v", err.Error())
		return TunnelConfig{}, err
	}

	host, _ := secs.GetKey("host")
	port := secs.Key("port").Validate(func(port string) string {
		if len(port) == 0 {
			return localClientDefaultPort
		}
		return port
	})

	envVars := secs.Key("env").ValueWithShadows()

	return TunnelConfig{
		host:    host.String(),
		port:    port,
		envVars: envVars,
	}, nil
}

// getHomePathIniFile Returns the full path of config file based on users home directory
func getHomePathIniFile(file string) string {
	home, _ := os.UserHomeDir()

	return home + "/" + file
}

// flagArguments Reads and parde the arguments passed to cfdtunnel.
// It returns an Argument Struct containing the profile, subcommand to run and all the arguments for the subcommand
func flagArguments() Arguments {
	profile := flag.String("profile", "", "Which cfdtunnel profile to use")
	version := flag.Bool("version", false, "Show cfdtunnel version")
	debug := flag.Bool("debug", false, "Enable Debug mode")

	flag.Parse()

	if *version {
		fmt.Println(appVersion)
		os.Exit(0)
	}

	if *debug {
		logLevel = log.DebugLevel
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

// checkSubCommandExists simple check if an specific binary exists in the OS
func checkSubCommandExists(command string) bool {
	_, err := exec.LookPath(command)
	if err != nil {
		log.Errorf("An error occurred: %v", err.Error())
		return false
	}

	return true
}
