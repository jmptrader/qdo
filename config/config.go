package config

type configuration struct {
	Defaultstore string
}

var Config *configuration

func newConfiguration() *configuration {
	return &configuration{}
}

func init() {
	Config = newConfiguration()
	Config.Defaultstore = "leveldb"
}
