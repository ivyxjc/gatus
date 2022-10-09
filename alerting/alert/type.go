package alert

// Type is the type of the alert.
// The value will generally be the name of the alert provider
type Type string

const (
	// TypeCustom is the Type for the custom alerting provider
	TypeCustom Type = "custom"

	// TypeDiscord is the Type for the discord alerting provider
	TypeDiscord Type = "discord"

	// TypeEmail is the Type for the email alerting provider
	TypeEmail Type = "email"

	// TypeGoogleChat is the Type for the googlechat alerting provider
	TypeGoogleChat Type = "googlechat"

	// TypeMatrix is the Type for the matrix alerting provider
	TypeMatrix Type = "matrix"

	// TypeMattermost is the Type for the mattermost alerting provider
	TypeMattermost Type = "mattermost"

	// TypeMessagebird is the Type for the messagebird alerting provider
	TypeMessagebird Type = "messagebird"

	// TypePagerDuty is the Type for the pagerduty alerting provider
	TypePagerDuty Type = "pagerduty"

	// TypeSlack is the Type for the slack alerting provider
	TypeSlack Type = "slack"

	TypeLark Type = "lark"

	// TypeTeams is the Type for the teams alerting provider
	TypeTeams Type = "teams"

	// TypeTelegram is the Type for the telegram alerting provider
	TypeTelegram Type = "telegram"

	// TypeTwilio is the Type for the twilio alerting provider
	TypeTwilio Type = "twilio"

	// TypeOpsgenie is the Type for the opsgenie alerting provider
	TypeOpsgenie Type = "opsgenie"
)
