package cfdtunnel

import (
	"os"
	"os/exec"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"gopkg.in/ini.v1"
)

const (
	localClientDefaultPort = "5555"
)

var (
	// LogLevel sets the level of each log
	LogLevel = log.WarnLevel
	// IniConfigFile sets the path of config file
	IniConfigFile = ".cfdtunnel/config"
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
	Profile string
	Command string
	Args    []string
}

type subCommand struct {
	*exec.Cmd
}

func init() {
	log.SetOutput(os.Stdout)
	log.SetLevel(LogLevel)
}

// Execute runs the entire flow of cfdtunnel tool
func (args Arguments) Execute() {
	log.SetLevel(LogLevel)

	config, err := readIniConfigFile(getHomePathIniFile(IniConfigFile))

	if err != nil {
		log.Fatalf("An error occurred reading your INI file: %v", err.Error())
	}

	tunnelConfig, err := config.readConfigSection(args.Profile)

	if err != nil {
		log.Fatalf("An error occurred reading your INI file: %v", err.Error())
	}

	cmd := tunnelConfig.startProxyTunnel()

	args.runSubCommand(tunnelConfig)

	// Kill it:
	commandKill(cmd)
}

// NewTunnel returns a new instance of the tunnel arguments to be executed
func NewTunnel(profile string, cmdArguments []string) *Arguments {

	return &Arguments{
		Profile: profile,
		Command: cmdArguments[0],
		Args:    cmdArguments[1:],
	}
}

// commandKill Kills an specific *exec.Cmd command
func commandKill(cmd *exec.Cmd) {
	log.Debugf("Trying to kill PID: %v", cmd.Process.Pid)

	if err := cmd.Process.Kill(); err != nil {
		log.Fatal("failed to kill process: ", err)
	}
}

// runSubCommand Runs the SubCommand and its arguments passed to cfdtunnel
func (args Arguments) runSubCommand(tunnelConfig TunnelConfig) {
	log.Debugf("Running subcommand: %v", args.Command)
	if !checkSubCommandExists(args.Command) {
		return
	}

	cmd := subCommand{exec.Command(args.Command, args.Args...)}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.setupEnvironmentVariables(tunnelConfig.envVars)

	err := cmd.Run()

	if err != nil {
		log.Errorf("An error occurred trying to run the command %v: %v", args.Command, err)
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
func (command subCommand) setupEnvironmentVariables(envVars []string) {

	command.Env = os.Environ()

	for _, env := range envVars {
		if !strings.Contains(env, "=") {
			continue
		}
		iniEnv := strings.Split(env, "=")
		log.Debugf("Exporting Environment variable: %v", env)
		command.Env = append(command.Env, iniEnv[0]+"="+iniEnv[1])
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

	envVars := []string{}
	if secs.Key("env").ValueWithShadows()[0] != "" {
		envVars = secs.Key("env").ValueWithShadows()
	}

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

// checkSubCommandExists simple check if an specific binary exists in the OS
func checkSubCommandExists(command string) bool {
	_, err := exec.LookPath(command)
	if err != nil {
		log.Errorf("An error occurred: %v", err.Error())
		return false
	}

	return true
}
