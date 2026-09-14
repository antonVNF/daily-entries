package config

type Config struct {
	Port        int
	DatabaseURl string
	LogLevel    string
}

func Load() *Config {

}

func getEnv() {
	
}
