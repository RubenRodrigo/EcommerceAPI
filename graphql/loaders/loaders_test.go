package loaders

import (
	"context"
	"sync"
	"testing"

	orderModels "github.com/rasadov/EcommerceAPI/order/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeOrdersClient struct {
	mu      sync.Mutex
	calls   int
	keys    []uint64
	results map[uint64][]orderModels.Order
}

func (client *fakeOrdersClient) GetOrdersForAccounts(_ context.Context, accountIDs []uint64) (map[uint64][]orderModels.Order, error) {
	client.mu.Lock()
	defer client.mu.Unlock()

	client.calls++
	client.keys = append([]uint64(nil), accountIDs...)
	return client.results, nil
}

func TestOrdersByAccountBatchesAndCachesLoads(t *testing.T) {
	client := &fakeOrdersClient{
		results: map[uint64][]orderModels.Order{
			1: {{ID: 10, AccountID: 1}},
			2: {{ID: 20, AccountID: 2}},
		},
	}
	loader := New(client).OrdersByAccount
	ctx := context.Background()

	accountOne := loader.LoadThunk(ctx, 1)
	accountTwo := loader.LoadThunk(ctx, 2)
	accountOneAgain := loader.LoadThunk(ctx, 1)

	ordersOne, err := accountOne()
	require.NoError(t, err)
	ordersTwo, err := accountTwo()
	require.NoError(t, err)
	ordersOneAgain, err := accountOneAgain()
	require.NoError(t, err)

	assert.Equal(t, uint(10), ordersOne[0].ID)
	assert.Equal(t, uint(20), ordersTwo[0].ID)
	assert.Equal(t, ordersOne, ordersOneAgain)
	assert.Equal(t, 1, client.calls)
	assert.ElementsMatch(t, []uint64{1, 2}, client.keys)
}

func TestFromContextRejectsMissingLoaders(t *testing.T) {
	requestLoaders, err := FromContext(context.Background())

	assert.Nil(t, requestLoaders)
	assert.EqualError(t, err, "dataloaders are missing from request context")
}
