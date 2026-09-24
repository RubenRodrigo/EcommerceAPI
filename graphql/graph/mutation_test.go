package graph

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPaymentMutationsRejectMissingInput(t *testing.T) {
	resolver := &mutationResolver{}

	t.Run("customer portal", func(t *testing.T) {
		result, err := resolver.CreateCustomerPortalSession(context.Background(), nil)

		assert.Nil(t, result)
		assert.EqualError(t, err, "credentials are required")
	})

	t.Run("checkout", func(t *testing.T) {
		result, err := resolver.CreateCheckoutSession(context.Background(), nil)

		assert.Nil(t, result)
		assert.EqualError(t, err, "checkout details are required")
	})
}
