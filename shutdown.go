package graceful

import (
	"context"

	"github.com/google/uuid"
)

// shutdown data struct that define for shutdown process
type shutdown struct {
	id      uuid.UUID
	tag     string
	process func(context.Context) error
}

// newShutdown init shutdown data using defined params
func newShutdown(tag string, process func(ctx context.Context) error) (shutdown, uuid.UUID) {
	id := uuid.New()

	if tag == "" {
		tag = id.String()
	}

	s := shutdown{
		id:      id,
		tag:     tag,
		process: process,
	}

	return s, id
}
