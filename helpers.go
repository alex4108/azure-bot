package main

import (
	"fmt"
	"reflect"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

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

func timestampFieldExists(obj *discordgo.MessageCreate) bool {
	metaValue := reflect.ValueOf(obj).Elem()
	field := metaValue.FieldByName("Timestamp")
	return field != (reflect.Value{})
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
