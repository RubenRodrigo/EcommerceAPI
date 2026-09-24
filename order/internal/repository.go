package internal

import (
	"context"
	"log"
	"time"

	"github.com/rasadov/EcommerceAPI/order/models"
	"gorm.io/gorm"
)

type Repository interface {
	Close()
	PutOrder(ctx context.Context, order *models.Order) error
	GetOrdersForAccount(ctx context.Context, accountId uint64) ([]*models.Order, error)
	GetOrdersForAccounts(ctx context.Context, accountIds []uint64) (map[uint64][]*models.Order, error)
	UpdateOrderPaymentStatus(ctx context.Context, orderId uint64, status string) error
}

type postgresRepository struct {
	db *gorm.DB
}

func NewPostgresRepository(db *gorm.DB) (Repository, error) {
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	err = sqlDB.Ping()
	if err != nil {
		return nil, err
	}

	err = db.AutoMigrate(&models.Order{}, &models.ProductsInfo{})
	if err != nil {
		return nil, err
	}

	return &postgresRepository{db}, nil
}

func (repository *postgresRepository) Close() {
	sqlDB, err := repository.db.DB()
	if err == nil {
		err = sqlDB.Close()
		if err != nil {
			log.Println("Error closing postgres repository")
			log.Println(err)
		}
	}
}

func (repository *postgresRepository) PutOrder(ctx context.Context, order *models.Order) error {
	tx := repository.db.WithContext(ctx).Begin()

	err := tx.WithContext(ctx).Create(&order).Error
	if err != nil {
		tx.Rollback()
		return err
	}

	for _, product := range order.Products {
		orderedProduct := models.ProductsInfo{
			OrderID:   order.ID,
			ProductID: product.ID,
			Quantity:  int(product.Quantity),
		}
		err = tx.Create(&orderedProduct).Error
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	if err = tx.Commit().Error; err != nil {
		return err
	}

	return nil
}

func (repository *postgresRepository) GetOrdersForAccount(ctx context.Context, accountId uint64) ([]*models.Order, error) {
	ordersByAccount, err := repository.GetOrdersForAccounts(ctx, []uint64{accountId})
	if err != nil {
		return nil, err
	}
	return ordersByAccount[accountId], nil
}

type orderProductRow struct {
	ID         uint
	CreatedAt  time.Time
	AccountID  uint64
	TotalPrice float64
	ProductID  string
	Quantity   int
}

func (repository *postgresRepository) GetOrdersForAccounts(ctx context.Context, accountIds []uint64) (map[uint64][]*models.Order, error) {
	ordersByAccount := make(map[uint64][]*models.Order, len(accountIds))
	for _, accountID := range accountIds {
		ordersByAccount[accountID] = []*models.Order{}
	}
	if len(accountIds) == 0 {
		return ordersByAccount, nil
	}

	var rows []orderProductRow
	err := repository.db.WithContext(ctx).
		Table("orders o").
		Select("o.id, o.created_at, o.account_id, o.total_price::money::numeric::float8 AS total_price, op.product_id, op.quantity").
		Joins("JOIN order_products op on o.id = op.order_id").
		Where("o.account_id IN ?", accountIds).
		Order("o.account_id, o.id").
		Scan(&rows).Error

	if err != nil {
		return nil, err
	}

	ordersByID := make(map[uint64]map[uint]*models.Order, len(accountIds))
	for _, row := range rows {
		accountOrders := ordersByID[row.AccountID]
		if accountOrders == nil {
			accountOrders = make(map[uint]*models.Order)
			ordersByID[row.AccountID] = accountOrders
		}

		order := accountOrders[row.ID]
		if order == nil {
			order = &models.Order{
				ID:         row.ID,
				CreatedAt:  row.CreatedAt,
				AccountID:  row.AccountID,
				TotalPrice: row.TotalPrice,
			}
			accountOrders[row.ID] = order
			ordersByAccount[row.AccountID] = append(ordersByAccount[row.AccountID], order)
		}

		order.Products = append(order.Products, &models.OrderedProduct{
			ID:       row.ProductID,
			Quantity: uint32(row.Quantity),
		})
	}

	return ordersByAccount, nil
}

func (repository *postgresRepository) UpdateOrderPaymentStatus(ctx context.Context, orderId uint64, status string) error {
	return repository.db.WithContext(ctx).Model(&models.Order{}).
		Where("id = ?", orderId).
		Update("payment_status", status).Error
}
