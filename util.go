package tax_receipt

import (
	"encoding/json"
	"os"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// CreateLocalizerBundle reads language files and registers them in i18n bundle

func Find(val string, slice []string) int {
	for i, item := range slice {
		if item == val {
			return i
		}
	}
	return -1
}
func fixDir() {
	fileInfo, err := os.Stat(ASSET_FOLDER)
	if err == nil && fileInfo.IsDir() {
		_ = os.Chdir(ASSET_FOLDER)
	}
}

func initLanguage(tag language.Tag, dict_file string) {

	byteValue, _ := os.ReadFile(dict_file)

	var translation map[string]string
	json.Unmarshal([]byte(byteValue), &translation)

	for key, mesg := range translation {
		message.SetString(tag, key, mesg)
	}
	// log.Printf("Loading %d translation from %s", len(translation), dict_file)
}

func Getenv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
