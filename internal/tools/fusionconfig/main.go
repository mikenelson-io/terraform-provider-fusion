package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
)

type fusionAuth struct {
	IssuerID       string `json:"issuer_id"`
	PrivatePEMFile string `json:"private_pem_file"`
}

type fusionProfile struct {
	Env      string
	Endpoint string
	Auth     fusionAuth
}

type fusionConfig struct {
	DefaultProfile string `json:"default_profile"`
	Profiles       map[string]fusionProfile
}

type sshParams struct {
	user   string
	target string
}

var sshTarget string
var containerUser string

func init() {
	flag.StringVar(&sshTarget, "target", "", "The remote target where the container is running")
	flag.StringVar(&containerUser, "container-user", "", "The user that deployed the control plane container")
}

func parseSSHParams() sshParams {
	defaultUser := "root"
	parts := strings.Split(sshTarget, "@")
	//TODO validation
	if len(parts) == 1 {
		return sshParams{
			user:   defaultUser,
			target: parts[0],
		}
	} else {
		return sshParams{
			user:   parts[0],
			target: parts[1],
		}
	}

}

func readPublicKey() ssh.AuthMethod {
	homeDir, err := os.UserHomeDir()
	checkErr(err)
	keyPath := filepath.Join(homeDir, ".ssh/id_rsa")
	b, err := ioutil.ReadFile(keyPath)
	checkErr(err)
	key, err := ssh.ParsePrivateKey(b)
	checkErr(err)
	return ssh.PublicKeys(key)
}

// SSH into the target and execute cmd, return output as a byte array
func (sshParams *sshParams) execRemoteCommand(cmd string) []byte {
	cfg := &ssh.ClientConfig{
		User:            sshParams.user,
		Auth:            []ssh.AuthMethod{readPublicKey()},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:22", sshParams.target), cfg)
	checkErr(err)

	sess, err := conn.NewSession()
	checkErr(err)

	defer sess.Close()

	outPipe, err := sess.StdoutPipe()
	checkErr(err)
	errPipe, err := sess.StderrPipe()
	checkErr(err)

	err = sess.Run(cmd)
	if err != nil {
		buf, _ := ioutil.ReadAll(errPipe) //TODO handle error
		fmt.Print(string(buf))
		panic(err)
	}

	buf, err := ioutil.ReadAll(outPipe)
	checkErr(err)
	return buf
}

func main() {
	flag.Parse()
	fmt.Println(sshTarget)

	//TODO validate parameters

	sshParams := parseSSHParams()
	remoteConfigPath := fmt.Sprintf("/home/%s/.pure/fusion.json", containerUser)

	homeDir, err := os.UserHomeDir()
	checkErr(err)
	localConfigPath := filepath.Join(homeDir, ".pure/fusion.json")
	localKeyPath := filepath.Join(homeDir, ".ssh/private.pem")

	// Retrieve the current fusion config
	// TODO docker exec {container} echo $HOME
	cmd := fmt.Sprintf("cat %s", remoteConfigPath)
	configData := sshParams.execRemoteCommand(cmd)

	// Deserialize the config
	var config fusionConfig
	err = json.Unmarshal(configData, &config)
	checkErr(err)

	//Check that there is only one running container
	//TODO

	//Get the name of the fusion container
	//TODO docker ps -q
	cmd = "docker ps | awk 'NR!=1 {print $1}'"
	container := string(sshParams.execRemoteCommand(cmd))
	// remove any trailing new lines
	container = strings.ReplaceAll(container, "\n", "")

	//Retrieve the value of the private key
	cmd = fmt.Sprintf("docker exec %s cat /tmp/private.pem", container)
	key := sshParams.execRemoteCommand(cmd)

	//Save the private Key
	err = os.WriteFile(localKeyPath, key, 0600)
	checkErr(err)

	//Create the new config
	profile := config.Profiles[config.DefaultProfile]
	profile.Endpoint = fmt.Sprintf("http://%s:8080", sshParams.target)
	profile.Auth.PrivatePEMFile = localKeyPath
	config.Profiles[config.DefaultProfile] = profile

	//Save the new config
	configJSON, err := json.Marshal(config)
	checkErr(err)
	err = os.WriteFile(localConfigPath, configJSON, 0644)
	checkErr(err)

	//TODO status messages
}

func checkErr(e error) {
	if e != nil {
		panic(e)
	}
}
