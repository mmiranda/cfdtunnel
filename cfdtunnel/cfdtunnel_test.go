package cfdtunnel

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"
	"testing"

	"log"

	"github.com/stretchr/testify/assert"
	"golang.org/x/net/proxy"
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
	config, _ := readIniConfigFile("../test/config")

	tunnelCfg, err := config.readConfigSection("alias1")

	assert.IsType(t, TunnelConfig{}, tunnelCfg)
	assert.Equal(t, "https://kubernetes.foo.bar.com", tunnelCfg.host)
	assert.Equal(t, "1234", tunnelCfg.port)
	assert.NoError(t, err)
}

func TestSectionDefaultPort(t *testing.T) {
	config, _ := readIniConfigFile("../test/config")

	tunnelCfg, err := config.readConfigSection("alias2")

	assert.IsType(t, TunnelConfig{}, tunnelCfg)
	assert.Equal(t, "sql.foo.bar.com", tunnelCfg.host)
	assert.Equal(t, localClientDefaultPort, tunnelCfg.port)
	assert.NoError(t, err)
}

func TestIniEnvVars(t *testing.T) {
	config, _ := readIniConfigFile("../test/config")

	tunnelCfg, err := config.readConfigSection("alias1")
	assert.Equal(t, []string{}, tunnelCfg.envVars)
	assert.NoError(t, err)

	tunnelCfg, err = config.readConfigSection("test-env-var")
	assert.Equal(t, []string{"MY_ENV_VAR=value"}, tunnelCfg.envVars)
	assert.NoError(t, err)

	tunnelCfg, err = config.readConfigSection("test-multi-env-var")
	assert.Equal(t, []string{"MY_ENV_VAR=value", "HTTPS_PROXY=socks5://127.0.0.1:5555", "IGNORED:value"}, tunnelCfg.envVars)
	assert.NoError(t, err)

}

func TestOSEnvVars(t *testing.T) {
	config, _ := readIniConfigFile("../test/config")
	tunnelCfg, _ := config.readConfigSection("test-multi-env-var")

	cmd := subCommand{exec.Command("ls")}

	cmd.setupEnvironmentVariables(tunnelCfg.envVars)

	assert.True(t, contains(cmd.Env, "MY_ENV_VAR=value"))
	assert.True(t, contains(cmd.Env, "HTTPS_PROXY=socks5://127.0.0.1:5555"))

	tunnelCfg, _ = config.readConfigSection("alias1")

	cmd = subCommand{exec.Command("ls")}
	cmd.setupEnvironmentVariables(tunnelCfg.envVars)

	assert.False(t, contains(cmd.Env, "MY_ENV_VAR=value"))

}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
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

func TestSubCommandExists(t *testing.T) {

	assert.True(t, checkSubCommandExists("echo"))
	assert.False(t, checkSubCommandExists("foobar"))

}

func TestConfigSectionDoesNotExists(t *testing.T) {
	config, _ := readIniConfigFile("../test/config")

	_, err := config.readConfigSection("missing")

	assert.Error(t, err)
}

func TestRunSubCommandStdOut(t *testing.T) {
	args := Arguments{
		Profile: "alias",
		Command: "ls",
		Args:    []string{"cfdtunnel.go"},
	}

	rescueStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	args.runSubCommand(TunnelConfig{})

	w.Close()
	out, _ := ioutil.ReadAll(r)
	os.Stdout = rescueStdout

	assert.Equal(t, "cfdtunnel.go\n\n", string(out))
}

func TestRunSubCommandMissing(t *testing.T) {

	// Run the crashing code when FLAG is set
	if os.Getenv("FLAG") == "1" {
		args := Arguments{
			Profile: "alias",
			Command: "lsssss",
			Args:    nil,
		}
		args.runSubCommand(TunnelConfig{})

		return
	}

	// Run the test in a subprocess
	cmd := exec.Command(os.Args[0], "-test.run=TestRunSubCommandMissing")
	cmd.Env = append(os.Environ(), "FLAG=1")
	err := cmd.Run()

	// Cast the error as *exec.ExitError and compare the result
	e, ok := err.(*exec.ExitError)
	assert.False(t, ok)
	assert.Nil(t, e)

}

func TestNewTunnel(t *testing.T) {

	tunnel := NewTunnel("test", []string{"cmd", "arg", "arg"})
	assert.IsType(t, &Arguments{}, tunnel)
	assert.Equal(t, "test", tunnel.Profile)
	assert.Equal(t, "cmd", tunnel.Command)
	assert.Equal(t, []string{"arg", "arg"}, tunnel.Args)

}

// TestSock5ProxyRunning launches the proxy tunnel and try to use it calling google.com
// If the result does not have "connection refused" we assume the proxy is running and responding
func TestSocks5ProxyRunning(t *testing.T) {
	tunnelConfig := TunnelConfig{"foo.bar", "1234", nil}
	cmd := tunnelConfig.startProxyTunnel()
	dialSocksProxy, err := proxy.SOCKS5("tcp", "127.0.0.1:1234", nil, proxy.Direct)
	if err != nil {
		fmt.Println("Error connecting to proxy:", err)
	}
	tr := &http.Transport{Dial: dialSocksProxy.Dial}

	// Create client
	myClient := &http.Client{
		Transport: tr,
	}

	_, err = myClient.Get("https://google.com")
	commandKill(cmd)
	assert.False(t, strings.Contains(err.Error(), "connect: connection refused"))

}
