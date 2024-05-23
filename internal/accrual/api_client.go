package accrual

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/PaBah/gofermart/internal/config"
	"github.com/PaBah/gofermart/internal/dto"
	"github.com/PaBah/gofermart/internal/logger"
	"github.com/PaBah/gofermart/internal/models"
	"github.com/PaBah/gofermart/internal/storage"
)

type OrdersAccrualClient struct {
	options *config.Options
	storage storage.Repository
}

var (
	ErrAccrualRequestCrashed     = errors.New("can not make request")
	ErrAccrualServiceServerError = errors.New("accrual service server error")
	ErrAccrualTooManyRequests    = errors.New("too many requests to accrual service")
	ErrAccrualNoData             = errors.New("unknown order number")
)

func (oac OrdersAccrualClient) ScrapeOrders() {
	go func() {
		for {
			ordersIDs, _ := oac.storage.GetAllOrdersIDs(context.Background())
			for _, orderID := range ordersIDs {
				order, err := oac.GetOrder(orderID)
				if err == nil {
					orderInstance := models.Order{
						Accrual: int(order.Accrual * 100),
						Number:  order.Order,
					}
					if order.Status == "REGISTERED" {
						orderInstance.Status = "NEW"
					} else {
						orderInstance.Status = order.Status
					}
					oac.storage.UpdateOrder(context.Background(), orderInstance)
				}
			}
		}
	}()
}

func (oac OrdersAccrualClient) GetOrder(number string) (order dto.AccrualOrderResponse, err error) {
	requestURL := fmt.Sprintf("%s/api/orders/%s", oac.options.AccrualSystemAddress, number)
	req, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return order, ErrAccrualRequestCrashed
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return order, ErrAccrualRequestCrashed
	}

	switch res.StatusCode {
	case http.StatusOK:
		resBody, err := io.ReadAll(res.Body)
		if err != nil {
			return order, ErrAccrualRequestCrashed
		}

		err = json.Unmarshal(resBody, &order)
		if err != nil {
			return order, ErrAccrualRequestCrashed
		}

		return order, nil
	case http.StatusInternalServerError:
		return order, ErrAccrualServiceServerError
	case http.StatusTooManyRequests:
		resBody, err := io.ReadAll(res.Body)
		if err != nil {
			return order, ErrAccrualTooManyRequests
		}
		logger.Log().Info(string(resBody))
	case http.StatusNoContent:
		return order, ErrAccrualNoData
	}
	defer res.Body.Close()

	return
}

func NewOrdersAccrualClient(options *config.Options, storage storage.Repository) OrdersAccrualClient {
	return OrdersAccrualClient{options: options, storage: storage}
}
