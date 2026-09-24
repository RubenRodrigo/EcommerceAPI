package graph

import (
	"context"
	"fmt"
	"log"
	"time"

	gql "github.com/99designs/gqlgen/graphql"
	"github.com/rasadov/EcommerceAPI/graphql/generated"
	"github.com/rasadov/EcommerceAPI/graphql/models"
	"github.com/rasadov/EcommerceAPI/graphql/utils"
	"github.com/rasadov/EcommerceAPI/pkg/auth"
	recommenderpb "github.com/rasadov/EcommerceAPI/recommender/generated/pb"
)

type queryResolver struct {
	server *Server
}

func (resolver *queryResolver) Accounts(
	ctx context.Context,
	pagination *generated.PaginationInput,
	id *int,
) ([]*models.Account, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if id != nil {
		res, err := resolver.server.accountClient.GetAccount(ctx, uint64(*id))
		if err != nil {
			log.Println(err)
			return nil, err
		}
		return []*models.Account{{
			ID:    uint64(res.ID),
			Name:  res.Name,
			Email: res.Email,
		}}, nil
	}

	skip, take := uint64(0), uint64(0)
	if pagination != nil {
		skip, take = utils.Bounds(pagination)
	}
	accountList, err := resolver.server.accountClient.GetAccounts(ctx, skip, take)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	var accounts []*models.Account
	for _, account := range accountList {
		account := &models.Account{
			ID:    uint64(account.ID),
			Name:  account.Name,
			Email: account.Email,
		}
		accounts = append(accounts, account)
	}
	return accounts, nil
}

func fieldSelected(ctx context.Context, name string, satisfies ...string) bool {
	for _, field := range gql.CollectFieldsCtx(ctx, satisfies) {
		if field.Name == name {
			return true
		}
	}
	return false
}

func (resolver *queryResolver) Product(
	ctx context.Context,
	pagination *generated.PaginationInput,
	query, id *string,
	viewedProductsIds []*string,
	byAccountId *bool,
) ([]*generated.Product, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	// Get single
	if id != nil {
		res, err := resolver.server.productClient.GetProduct(ctx, *id)
		if err != nil {
			log.Println(err)
			return nil, err
		}
		return []*generated.Product{{
			ID:          res.ID,
			Name:        res.Name,
			Description: res.Description,
			Price:       res.Price,
			AccountID:   res.AccountID,
		}}, nil
	}
	skip, take := uint64(0), uint64(0)
	if pagination != nil {
		skip, take = utils.Bounds(pagination)
	}

	// Get recommendations
	if viewedProductsIds != nil {
		productIds := make([]string, len(viewedProductsIds))
		for i, id := range viewedProductsIds {
			productIds[i] = *id
		}
		res, err := resolver.server.recommenderClient.GetRecommendationBasedOnViewed(ctx, productIds, skip, take)
		if err != nil {
			log.Println(err)
			return nil, err
		}
		productList := res.GetRecommendedProducts()
		accountIDs := map[string]int{}
		if fieldSelected(ctx, "accountId", "Product") {
			accountIDs, err = resolver.productAccountIDs(ctx, productList)
			if err != nil {
				return nil, err
			}
		}
		var products []*generated.Product
		for _, product := range productList {
			products = append(products,
				&generated.Product{
					ID:          product.Id,
					Name:        product.Name,
					Description: product.Description,
					Price:       product.Price,
					AccountID:   accountIDs[product.Id],
				},
			)
		}
		return products, nil
	}

	if byAccountId != nil && *byAccountId {
		accountId, err := auth.GetUserId(ctx)
		if err != nil {
			return nil, err
		}
		skip = 0
		take = 100
		res, err := resolver.server.recommenderClient.GetRecommendationForUser(ctx, accountId, skip, take)
		if err != nil {
			log.Println(err)
			return nil, err
		}
		productList := res.GetRecommendedProducts()
		accountIDs := map[string]int{}
		if fieldSelected(ctx, "accountId", "Product") {
			accountIDs, err = resolver.productAccountIDs(ctx, productList)
			if err != nil {
				return nil, err
			}
		}
		var products []*generated.Product
		for _, product := range productList {
			products = append(products,
				&generated.Product{
					ID:          product.Id,
					Name:        product.Name,
					Description: product.Description,
					Price:       product.Price,
					AccountID:   accountIDs[product.Id],
				},
			)
		}
		return products, nil
	}

	q := ""
	if query != nil {
		q = *query
	}
	productList, err := resolver.server.productClient.GetProducts(ctx, skip, take, nil, q)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	var products []*generated.Product
	for _, product := range productList {
		products = append(products,
			&generated.Product{
				ID:          product.ID,
				Name:        product.Name,
				Description: product.Description,
				Price:       product.Price,
				AccountID:   product.AccountID,
			},
		)
	}

	return products, nil
}

func (resolver *queryResolver) productAccountIDs(ctx context.Context, recommendations []*recommenderpb.ProductReplica) (map[string]int, error) {
	ids := make([]string, 0, len(recommendations))
	for _, product := range recommendations {
		ids = append(ids, product.GetId())
	}
	if len(ids) == 0 {
		return map[string]int{}, nil
	}

	products, err := resolver.server.productClient.GetProducts(ctx, 0, uint64(len(ids)), ids, "")
	if err != nil {
		return nil, err
	}
	accountIDs := make(map[string]int, len(products))
	for _, product := range products {
		accountIDs[product.ID] = product.AccountID
	}
	for _, product := range recommendations {
		if _, ok := accountIDs[product.GetId()]; !ok {
			return nil, fmt.Errorf("product %q not found while resolving accountId", product.GetId())
		}
	}
	return accountIDs, nil
}
