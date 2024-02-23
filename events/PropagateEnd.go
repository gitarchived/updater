package events

import (
	"errors"
	"os"

	"github.com/go-resty/resty/v2"
)

func PropagateEnd(total int, updated int) error {
	url := os.Getenv("EVENTS_END")
	key := os.Getenv("EVENTS_KEY")

	client := resty.New()

	resp, err := client.R().
		SetHeader("Authorization", key).
		SetHeader("Content-Type", "application/json").
		SetBody(map[string]int{"total": total, "updated": updated}).
		Post(url)

	if err != nil {
		return err
	}

	if resp.StatusCode() != 200 {
		return errors.New("Error propagating end event")
	}

	return nil
}
