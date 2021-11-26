package main

import (
	"log"
	"os"
	"os/exec"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/ini.v1"
)

const (
	iniFile = ".cfdtunnel/config-test"
)

func TestAbsolutePathIniFile(t *testing.T) {
	home, _ := os.UserHomeDir()
	assert.Equal(t, home+"/"+iniFile, getHomePathIniFile(iniFile))
}

func TestIniFileExists(t *testing.T) {
	iniFile := getHomePathIniFile(iniFile)
	// Make sure file is gone before test
	os.Remove(iniFile)

	_, err := readIniConfigFile(iniFile)

	assert.NoFileExists(t, iniFile)
	assert.Error(t, err)

	helperCreateFile()

	config, err := readIniConfigFile(iniFile)

	assert.IsType(t, &ini.File{}, config.ini)
	assert.NoError(t, err)
	assert.FileExists(t, iniFile)

	os.Remove(iniFile)

}

func helperCreateFile() {

	iniConfig := getHomePathIniFile(iniFile)
	err := os.MkdirAll(path.Dir(iniConfig), os.ModePerm)
	if err != nil {
		log.Fatalf("Failed to create directory: %v", err.Error())
	}
	_, err = os.OpenFile(iniConfig, os.O_RDONLY|os.O_CREATE, 0666)

	if err != nil {
		log.Fatalf("Error creating file: %v", err.Error())
	}
}

func TestSectionComplete(t *testing.T) {
	config, _ := readIniConfigFile("test/config")

	tunnelCfg, err := config.readConfigSection("alias1")

	assert.IsType(t, TunnelConfig{}, tunnelCfg)
	assert.Equal(t, "https://kubernetes.foo.bar.com", tunnelCfg.host)
	assert.Equal(t, "1234", tunnelCfg.port)
	assert.NoError(t, err)
}

func TestSectionDefaultPort(t *testing.T) {
	config, _ := readIniConfigFile("test/config")

	tunnelCfg, err := config.readConfigSection("alias2")

	assert.IsType(t, TunnelConfig{}, tunnelCfg)
	assert.Equal(t, "sql.foo.bar.com", tunnelCfg.host)
	assert.Equal(t, localClientDefaultPort, tunnelCfg.port)
	assert.NoError(t, err)
}

func TestIniEnvVars(t *testing.T) {
	config, _ := readIniConfigFile("test/config")

	tunnelCfg, err := config.readConfigSection("test-env-var")
	assert.Equal(t, []string{"MY_ENV_VAR=value"}, tunnelCfg.envVars)
	assert.NoError(t, err)

	tunnelCfg, err = config.readConfigSection("test-multi-env-var")
	assert.Equal(t, []string{"MY_ENV_VAR=value", "HTTPS_PROXY=127.0.0.1:5555"}, tunnelCfg.envVars)
	assert.NoError(t, err)

}

func TestOSEnvVars(t *testing.T) {
	config, _ := readIniConfigFile("test/config")
	tunnelCfg, _ := config.readConfigSection("test-multi-env-var")

	tunnelCfg.setupEnvironmentVariables()

	assert.Equal(t, "value", os.Getenv("MY_ENV_VAR"))
	assert.Equal(t, "127.0.0.1:5555", os.Getenv("HTTPS_PROXY"))

}

func TestProxyTunnel(t *testing.T) {
	tunnelConfig := TunnelConfig{"foo.bar", "1234", nil}
	cmd := tunnelConfig.startProxyTunnel()
	osPid, _ := os.FindProcess(cmd.Process.Pid)
	assert.Equal(t, cmd.Process.Pid, osPid.Pid)
	commandKill(cmd)

}

func TestTunnelSamePort(t *testing.T) {

	tunnelCfg := TunnelConfig{"foo.bar.first", "1234", nil}
	cmd1 := tunnelCfg.startProxyTunnel()

	tunnelCfg = TunnelConfig{"foo.bar.first", "1234", nil}
	cmd2 := tunnelCfg.startProxyTunnel()

	err := cmd2.Wait()
	assert.Error(t, err)

	commandKill(cmd1)
}

// TestFlagsEnptyArguments uses the approach from Sartaj Singh on his article: https://sr-taj.medium.com/how-to-test-methods-that-kill-your-program-in-golang-e3b879185b8a
// What we want to test here is the os.Exit(1) when flagArguments() fail with empty arguments
func TestFlagsEmptyArguments(t *testing.T) {
	// Run the crashing code when FLAG is set
	if os.Getenv("FLAG") == "1" {
		// Calls flagArguments withour manipulate os.Args
		_ = flagArguments()
		return
	}

	// Run the test in a subprocess
	cmd := exec.Command(os.Args[0], "-test.run=TestFlagsEmptyArguments")
	cmd.Env = append(os.Environ(), "FLAG=1")
	err := cmd.Run()

	// Cast the error as *exec.ExitError and compare the result
	e, ok := err.(*exec.ExitError)
	assert.Equal(t, true, ok)
	assert.Equal(t, "exit status 1", e.Error())
}

func TestFlagArguments(t *testing.T) {

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"rootTest", "--profile", "alias", "subcommand", "arg1", "arg2"}
	args := flagArguments()

	assert.Equal(t, "alias", *args.profile)
	assert.Equal(t, "subcommand", args.command)
	assert.Equal(t, []string{"arg1", "arg2"}, args.args)
}

func TestSubCommandExists(t *testing.T) {

	assert.True(t, checkSubCommandExists("echo"))
	assert.False(t, checkSubCommandExists("foobar"))

}

func TestConfigSectionDoesNotExists(t *testing.T) {
	config, _ := readIniConfigFile("test/config")

	_, err := config.readConfigSection("missing")

	assert.Error(t, err)
}
