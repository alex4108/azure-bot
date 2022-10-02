package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute"
	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

var Cfg *Config

type ConfigEntry struct {
	VMName         string `yaml:"azurevm_name"`
	LogicalName    string `yaml:"logical_name"`
	ResourceGroup  string `yaml:"resource_group"`
	SubscriptionId string `yaml:"subscription_id"`
}

type Config struct {
	Servers []ConfigEntry `yaml:"vms"`
}

func main() {
	log.Info("Initializing...")
	log.SetReportCaller(true)
	log.SetLevel(log.DebugLevel)

	InCI, InCIExist := os.LookupEnv("CI")
	if InCIExist && InCI == "true" {
		log.Info("Running in CI.  This proves functionality?")
		os.Exit(0)
	}

	Token, tokenExists := os.LookupEnv("AZURE_BOT_DISCORD_TOKEN")
	if !tokenExists {
		log.Error("AZURE_BOT_DISCORD_TOKEN is not set.  Exiting.")
		os.Exit(2)
	}

	CfgPath, err := ParseFlags()
	if err != nil {
		log.Fatal(err)
	}
	err = NewConfig(CfgPath)
	if err != nil {
		log.Fatal(err)
	}

	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + Token)
	if err != nil {
		log.Error("error creating Discord session,", err)
		os.Exit(3)
	}

	dg.AddHandler(messageCreate)

	dg.Identify.Intents = discordgo.IntentsAllWithoutPrivileged | discordgo.IntentsGuildMembers | discordgo.IntentMessageContent

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		log.Error("Error opening websocket connection,", err)
		os.Exit(4)
	}

	// Wait here until CTRL-C or other term signal is received.
	log.Info("azure-bot is online!")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// Cleanly close down the Discord session.
	dg.Close()
}

// messageCreate handles new message events from Discord and routes them to the appropriate handler.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	// Ignore all messages created by the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	if strings.Split(m.Content, " ")[0] == "$startvm" {
		go startCommand(s, m)
	} else if strings.Split(m.Content, " ")[0] == "$stopvm" {
		go stopCommand(s, m)
	} else if m.Content == "$help" {
		go helpCommand(s, m)
	} else if m.Content == "$ping" {
		go pingCommand(s, m)
	}
}

func getResourceGroupFromConfig(targetVM string) (string, error) {
	for _, v := range Cfg.Servers {
		if v.LogicalName == targetVM {
			return v.ResourceGroup, nil
		}
	}
	return "", fmt.Errorf("failed to find a VM with that Name")
}

func getVMNameFromConfig(targetVM string) (string, error) {
	for _, v := range Cfg.Servers {
		if v.LogicalName == targetVM {
			return v.VMName, nil
		}
	}
	return "", fmt.Errorf("failed to find a VM with that Name")
}

func getSubscriptionIdFromConfig(targetVM string) (string, error) {
	for _, v := range Cfg.Servers {
		if v.LogicalName == targetVM {
			return v.SubscriptionId, nil
		}
	}
	return "", fmt.Errorf("failed to find a VM with that Name")
}

// startCommand is the command handler to start an azure VM.
func startCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	targetVM, err := getTargetVMFromCommand(m.Content)
	if err != nil {
		response := fmt.Sprintf("Failed to stop the VM: %v", err)
		respond(s, m.ChannelID, response)
		return
	}

	log.Infof("Received start command for VM: %v", targetVM)

	resourceGroup, err := getResourceGroupFromConfig(targetVM)
	if err != nil {
		response := fmt.Sprintf("Failed to start the VM: %v", err)
		respond(s, m.ChannelID, response)
		return
	}

	vmName, err := getVMNameFromConfig(targetVM)
	if err != nil {
		response := fmt.Sprintf("Failed to start the VM: %v", err)
		respond(s, m.ChannelID, response)
		return
	}

	subscriptionId, err := getSubscriptionIdFromConfig(targetVM)
	if err != nil {
		response := fmt.Sprintf("Failed to start the VM: %v", err)
		respond(s, m.ChannelID, response)
		return
	}

	respond(s, m.ChannelID, "Starting VM "+targetVM+"...")
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Fatalf("failed to obtain a credential: %v", err)
		respondError(s, m.ChannelID)
		return
	}

	client, err := armcompute.NewVirtualMachinesClient(subscriptionId, cred, nil)
	if err != nil {
		log.Errorf("failed to create VM client: %v", err)
		respondError(s, m.ChannelID)
		return
	}
	poller, err := client.BeginStart(
		context.Background(),
		resourceGroup,
		vmName,
		nil,
	)

	if err != nil {
		log.Errorf("failed to obtain a response: %v", err)
		respondError(s, m.ChannelID)
		return
	}

	_, err = poller.PollUntilDone(context.Background(), nil)
	if err != nil {
		log.Errorf("failed to start the vm: %v", err)
		response := fmt.Sprintf("Failed to start the VM: %v", err)
		respond(s, m.ChannelID, response)
		return
	}

	log.Infof("VM %v started.", targetVM)
	respond(s, m.ChannelID, "VM "+targetVM+" started.")
}

func getTargetVMFromCommand(content string) (string, error) {
	if len(strings.Split(content, " ")) < 2 {
		return "", fmt.Errorf("no VM name provided")
	}

	if len(strings.Split(content, " ")) > 2 {
		return "", fmt.Errorf("too many arguments")
	}

	targetVM := strings.Split(content, " ")[1]
	return targetVM, nil
}

// stopCommand is the command handler to stop an azure VM.
func stopCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	targetVM, err := getTargetVMFromCommand(m.Content)
	if err != nil {
		response := fmt.Sprintf("Failed to stop the VM: %v", err)
		respond(s, m.ChannelID, response)
		return
	}

	log.Infof("Received stop command for VM: %v", targetVM)

	resourceGroup, err := getResourceGroupFromConfig(targetVM)
	if err != nil {
		response := fmt.Sprintf("Failed to stop the VM: %v", err)
		respond(s, m.ChannelID, response)
		return
	}

	vmName, err := getVMNameFromConfig(targetVM)
	if err != nil {
		response := fmt.Sprintf("Failed to stop the VM: %v", err)
		respond(s, m.ChannelID, response)
		return
	}

	subscriptionId, err := getSubscriptionIdFromConfig(targetVM)
	if err != nil {
		response := fmt.Sprintf("Failed to stop the VM: %v", err)
		respond(s, m.ChannelID, response)
		return
	}

	respond(s, m.ChannelID, "Stopping VM "+targetVM+"...")
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Errorf("failed to obtain a credential: %v", err)
		respondError(s, m.ChannelID)
		return
	}

	client, err := armcompute.NewVirtualMachinesClient(subscriptionId, cred, nil)
	if err != nil {
		log.Errorf("failed to create VM client: %v", err)
		respondError(s, m.ChannelID)
		return
	}
	poller, err := client.BeginDeallocate(
		context.Background(),
		resourceGroup,
		vmName,
		nil,
	)

	if err != nil {
		log.Errorf("failed to obtain a response: %v", err)
		respondError(s, m.ChannelID)
		return
	}

	_, err = poller.PollUntilDone(context.Background(), nil)
	if err != nil {
		log.Errorf("failed to stop the vm: %v", err)
		response := fmt.Sprintf("Failed to stop the VM: %v", err)
		respond(s, m.ChannelID, response)
		return
	}

	log.Infof("VM %v stopped.", targetVM)
	respond(s, m.ChannelID, "VM "+targetVM+" stopped.")
}

func timestampFieldExists(obj *discordgo.MessageCreate) bool {
	metaValue := reflect.ValueOf(obj).Elem()
	field := metaValue.FieldByName("Timestamp")
	return field != (reflect.Value{})
}

// pingCommand is the command handler to ping the bot.
func pingCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	now := time.Now()
	latency := ""
	if timestampFieldExists(m) {
		diff := m.Timestamp.Sub(now)
		latency = "(" + strconv.Itoa(int(diff.Milliseconds())) + " ms)"
	}
	respond(s, m.ChannelID, "Pong! "+latency)
}

// helpCommand is the command handler to show the help message.
func helpCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	messageContent := `azure-bot, an open source Discord Bot.

Available Commands:
	$stopvm <vm_name>
	$startvm <vm_name>
	
Proudly maintained by Alex https://github.com/alex4108/azure-bot
`

	respond(s, m.ChannelID, messageContent)
}

// respondError is a quick way to respond with an error message.
func respondError(s *discordgo.Session, channelID string) {
	respond(s, channelID, "An internal error occured.  Please raise a bug on the github repository for further investigation.")
}

func respond(s *discordgo.Session, channelID string, response string) {
	_, err := s.ChannelMessageSend(channelID, response)
	if err != nil {
		log.Errorf("Failed to respond to setApproverChannelCommand command. %s", err)
	}
}

// NewConfig returns a new decoded Config struct
func NewConfig(configPath string) error {
	// Create config structure
	config := &Config{}

	// Open config file
	file, err := os.Open(configPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Init new YAML decode
	d := yaml.NewDecoder(file)

	// Start YAML decoding from file
	if err := d.Decode(&config); err != nil {
		return err
	}

	Cfg = config
	return nil
}

// ValidateConfigPath just makes sure, that the path provided is a file,
// that can be read
func ValidateConfigPath(path string) error {
	s, err := os.Stat(path)
	if err != nil {
		return err
	}
	if s.IsDir() {
		return fmt.Errorf("'%s' is a directory, not a normal file", path)
	}
	return nil
}

// ParseFlags will create and parse the CLI flags
// and return the path to be used elsewhere
func ParseFlags() (string, error) {
	// String that contains the configured configuration path
	var configPath string

	// Set up a CLI flag called "-config" to allow users
	// to supply the configuration file
	flag.StringVar(&configPath, "config", "/workspace/azure-bot-config.yml", "path to config file")

	// Actually parse the flags
	flag.Parse()

	envPath, envPathExist := os.LookupEnv("CONFIG_PATH")
	if envPathExist {
		configPath = envPath
	}

	// Validate the path first
	if err := ValidateConfigPath(configPath); err != nil {
		return "", err
	}

	// Return the configuration path
	return configPath, nil
}
