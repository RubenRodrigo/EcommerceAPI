package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/rasadov/EcommerceAPI/pkg/contextkeys"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetUserIDInt(t *testing.T) {
	t.Run("returns the authenticated user ID", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), contextkeys.UserIDKey, uint64(42))

		userID, err := GetUserIdInt(ctx)

		require.NoError(t, err)
		assert.Equal(t, 42, userID)
	})

	t.Run("returns an error instead of panicking when unauthenticated", func(t *testing.T) {
		assert.NotPanics(t, func() {
			userID, err := GetUserIdInt(context.Background())

			assert.Zero(t, userID)
			assert.ErrorIs(t, err, ErrUnauthorized)
			assert.EqualError(t, err, "unauthorized")
		})
	})
}

func TestGetUserID(t *testing.T) {
	t.Run("returns the authenticated user ID as a string", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), contextkeys.UserIDKey, uint64(42))

		userID, err := GetUserId(ctx)

		require.NoError(t, err)
		assert.Equal(t, "42", userID)
	})

	t.Run("returns the standard unauthorized error", func(t *testing.T) {
		userID, err := GetUserId(context.Background())

		assert.Empty(t, userID)
		assert.True(t, errors.Is(err, ErrUnauthorized))
	})
}
