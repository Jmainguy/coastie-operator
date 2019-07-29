package coastieservice

import (
	"fmt"
	"github.com/nlopes/slack"
)

func notifySlack(token, channelID, message string) {
	api := slack.New(token)
	params := slack.PostMessageParameters{}
	params.LinkNames = 1

	_, _, err := api.PostMessage(channelID, slack.MsgOptionText(message, false), slack.MsgOptionPostMessageParameters(params))
	fmt.Println(err)
	fmt.Println(message)
}
