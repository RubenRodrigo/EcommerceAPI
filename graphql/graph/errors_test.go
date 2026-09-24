package graph

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/rasadov/EcommerceAPI/pkg/auth"
	"github.com/stretchr/testify/assert"
)

func TestErrorPresenterStandardizesUnauthorizedErrors(t *testing.T) {
	err := fmt.Errorf("could not resolve user: %w", auth.ErrUnauthorized)

	presented := ErrorPresenter(context.Background(), err)

	assert.Equal(t, "unauthorized", presented.Message)
	assert.Equal(t, ErrorCodeUnauthenticated, presented.Extensions["code"])
}

func TestErrorPresenterPreservesOtherErrors(t *testing.T) {
	err := errors.New("downstream unavailable")

	presented := ErrorPresenter(context.Background(), err)

	assert.Equal(t, err.Error(), presented.Message)
	assert.NotEqual(t, ErrorCodeUnauthenticated, presented.Extensions["code"])
}
