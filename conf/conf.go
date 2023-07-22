package conf

import (
	"github.com/pelletier/go-toml/v2"
	"os"
)

type ConfigStruct struct {
	Port   int      `toml:"port"`
	Tokens []string `toml:"tokens"`
	//Gateway       string   `toml:"gateway"`
	Bot           map[string]string `toml:"bot"`
	SimulateRoles int               `toml:"simulate-roles"`
	RateLimit     int               `toml:"rate-limit"`
	CoolDown      int               `toml:"cool-down"`
	AutoReload    int               `toml:"auto-reload"`
	Timeout       int               `toml:"timeout"`
	ApiTimeout    int               `toml:"api-timeout"`
	Proxy         string            `toml:"proxy"`
}

type ModelDef struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int    `json:"created"`
	OwnedBy string `json:"owned_by"`
}

type ModelsResp struct {
	Object string     `json:"object"`
	Data   []ModelDef `json:"data"`
}

//func (c ConfigStruct) GetGatewayWsURL() string {
//	str := strings.ReplaceAll(c.Gateway, "http://", "ws://")
//	str = strings.ReplaceAll(str, "https://", "wss://")
//	return str
//}

var Conf ConfigStruct

var Models ModelsResp

func Setup() {
	v, err := os.ReadFile("config.toml")
	if err != nil {
		panic(err)
	}
	err = toml.Unmarshal(v, &Conf)
	if err != nil {
		panic(err)
	}
	if Conf.Port == 0 {
		Conf.Port = 3701
	}
	if Conf.RateLimit == 0 {
		Conf.RateLimit = 10
	}
	if Conf.Bot == nil {
		Conf.Bot = map[string]string{
			"gpt-3.5-turbo":      "ChatGPT",
			"gpt-3.5-turbo-0301": "ChatGPT",
			"gpt-4":              "GPT-4",
			"gpt-4-0314":         "GPT-4",
			"gpt-4-32k":          "GPT-4-32k",
			"claude-2-100k":      "Claude-2-100k",
		}
	}

	//if Conf.Proxy == "" {
	//	Conf.Proxy = nil
	//}

	Models.Object = ""

	for key := range Conf.Bot {
		Models.Data = append(Models.Data, ModelDef{
			ID:      key,
			Object:  "",
			Created: 0,
			OwnedBy: "",
		})
	}
}
