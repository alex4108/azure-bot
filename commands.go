package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute"
	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

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

func stateCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	targetVM, err := getTargetVMFromCommand(m.Content)
	if err != nil {
		response := fmt.Sprintf("Failed to stop the VM: %v", err)
		respond(s, m.ChannelID, response)
		return
	}

	log.Infof("Received state command for VM: %v", targetVM)

	resourceGroup, err := getResourceGroupFromConfig(targetVM)
	if err != nil {
		response := fmt.Sprintf("Failed to get state for the VM: %v", err)
		respond(s, m.ChannelID, response)
		return
	}

	vmName, err := getVMNameFromConfig(targetVM)
	if err != nil {
		response := fmt.Sprintf("Failed to get state for the VM: %v", err)
		respond(s, m.ChannelID, response)
		return
	}

	subscriptionId, err := getSubscriptionIdFromConfig(targetVM)
	if err != nil {
		response := fmt.Sprintf("Failed to get state for the VM: %v", err)
		respond(s, m.ChannelID, response)
		return
	}

	respond(s, m.ChannelID, fmt.Sprintf("Getting State for VM %v...", targetVM))
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
	ctx := context.Background()
	res, err := client.InstanceView(ctx,
		resourceGroup,
		vmName,
		nil)
	if err != nil {
		log.Fatalf("failed to finish the request: %v", err)
	}

	log.Infof("Got info for VM %v.", targetVM)
	status := ""
	for k, r := range res.Statuses {
		if k == 0 {
			status = fmt.Sprintf(status+"Provisioning State: %v | ", *r.DisplayStatus)
		} else if k == 1 {
			status = fmt.Sprintf(status+"Running State: %v", *r.DisplayStatus)
		}
	}
	respond(s, m.ChannelID, fmt.Sprintf("VM Info: \n%v", status))
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
	$stopvm <vm_name> Stops a VM
	$startvm <vm_name> Starts a VM
	$vmstate <vm_name> Shows state of the VM.
	$ping
	
Proudly maintained by Alex https://github.com/alex4108/azure-bot
`

	respond(s, m.ChannelID, messageContent)
}
