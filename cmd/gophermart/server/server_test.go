package server

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/PaBah/gofermart/internal/auth"
	"github.com/PaBah/gofermart/internal/config"
	"github.com/PaBah/gofermart/internal/mock"
	"github.com/PaBah/gofermart/internal/models"
	"github.com/PaBah/gofermart/internal/storage"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestServer(t *testing.T) {
	// описываем набор данных: метод запроса, ожидаемый код ответа, ожидаемое тело
	testCases := []struct {
		method       string
		requestBody  string
		contentType  string
		path         string
		userID       string
		expectedCode int
		expectedBody string
		storage      storage.Repository
	}{
		{method: http.MethodPost, contentType: "application/json", path: "/api/user/register", requestBody: `{"login":"test","password":"test"}`, expectedCode: http.StatusOK},
		{method: http.MethodPost, contentType: "application/json", path: "/api/user/login", requestBody: `{"login":"test","password":"test"}`, expectedCode: http.StatusUnauthorized},
		//Order Registration
		{method: http.MethodPost, path: "/api/user/orders", contentType: "text/plain", userID: "test", requestBody: "3081279352", expectedCode: http.StatusOK},
		{method: http.MethodPost, path: "/api/user/orders", contentType: "text/plain", userID: "test", requestBody: "12345678903", expectedCode: http.StatusAccepted},
		{method: http.MethodPost, path: "/api/user/orders", requestBody: "12345678903", userID: "test", expectedCode: http.StatusBadRequest},
		{method: http.MethodPost, path: "/api/user/orders", requestBody: "12345678903", expectedCode: http.StatusUnauthorized},
		{method: http.MethodPost, path: "/api/user/orders", contentType: "text/plain", userID: "test", requestBody: "6400700313", expectedCode: http.StatusConflict},
		{method: http.MethodPost, path: "/api/user/orders", contentType: "text/plain", userID: "test", requestBody: "123", expectedCode: http.StatusUnprocessableEntity},
		//Orders List
		{method: http.MethodGet, path: "/api/user/orders", contentType: "application/json", userID: "test", expectedCode: http.StatusOK, expectedBody: `[{"number":"12345678903","status":"NEW","uploaded_at":"2020-12-10T15:15:45+03:00"}]`},
		{method: http.MethodGet, path: "/api/user/orders", contentType: "application/json", userID: "test2", expectedCode: http.StatusNoContent},
		{method: http.MethodGet, path: "/api/user/orders", contentType: "application/json", expectedCode: http.StatusUnauthorized},
		{method: http.MethodGet, path: "/api/user/orders", contentType: "application/json", userID: "test", expectedCode: http.StatusInternalServerError},
		//Get Balance
		{method: http.MethodGet, path: "/api/user/balance", contentType: "application/json", userID: "test", expectedCode: http.StatusOK, expectedBody: `{"current":500.5,"withdrawn":42}`},
		{method: http.MethodGet, path: "/api/user/balance", contentType: "application/json", expectedCode: http.StatusUnauthorized},
		//{method: http.MethodGet, path: "/api/user/balance", contentType: "application/json", userID: "test", expectedCode: http.StatusOK, expectedBody: `{"current":500.5,"withdrawn":42}`},

		{method: http.MethodPost, path: "/api/user/balance/withdraw", contentType: "application/json", userID: "test", requestBody: `{"order": "2377225624","sum":123}`, expectedCode: http.StatusOK},
		{method: http.MethodGet, path: "/api/user/withdrawals", contentType: "application/json", userID: "test", expectedCode: http.StatusOK, expectedBody: `[{"order":"2377225624","sum":123,"processed_at":"2020-12-09T16:09:57+03:00"}]`},
	}

	options := &config.Options{
		RunAddress:           ":8081",
		DatabaseURI:          "http://localhost:8080",
		AccrualSystemAddress: "wrong DSN",
	}
	uploadedAt, _ := time.Parse(time.RFC3339, "2020-12-10T15:15:45+03:00")
	processedAt, _ := time.Parse(time.RFC3339, "2020-12-09T16:09:57+03:00")

	var store storage.Repository
	ctrl := gomock.NewController(t)
	rm := mock.NewMockRepository(ctrl)
	store = rm

	rm.
		EXPECT().
		CreateUser(gomock.Any(), gomock.Any()).
		Return(models.User{ID: "test", Login: "test", Password: "test"}, nil).
		AnyTimes()
	rm.
		EXPECT().
		AuthorizeUser(gomock.Any(), gomock.Any()).
		Return(models.User{ID: "test", Login: "test", Password: "test"}, nil).
		AnyTimes()
	rm.
		EXPECT().
		RegisterOrder(gomock.Any(), "12345678903").
		Return(models.Order{Number: "12345678903", UserID: "test", Status: "NEW", Accrual: 0}, nil).
		AnyTimes()
	rm.
		EXPECT().
		RegisterOrder(gomock.Any(), "3081279352").
		Return(models.Order{Number: "3081279352", UserID: "test", Status: "NEW", Accrual: 0}, errors.New("already exists")).
		AnyTimes()
	rm.
		EXPECT().
		RegisterOrder(gomock.Any(), "6400700313").
		Return(models.Order{Number: "6400700313", UserID: "not_test", Status: "NEW", Accrual: 0}, errors.New("already exists")).
		AnyTimes()
	rm.
		EXPECT().
		GetUsersOrders(gomock.Any()).
		Return([]models.Order{models.Order{Number: "12345678903", UserID: "test", Status: "NEW", Accrual: 0, UploadedAt: uploadedAt}}, nil).
		Times(1)
	rm.
		EXPECT().
		GetUsersOrders(gomock.Any()).
		Return([]models.Order{}, nil).
		Times(1)
	rm.
		EXPECT().
		GetUsersOrders(gomock.Any()).
		Return([]models.Order{}, errors.New("DB brake down")).
		Times(1)
	rm.
		EXPECT().
		GetUsersBalance(gomock.Any()).
		Return(542.5, nil).
		AnyTimes()
	rm.
		EXPECT().
		GetUsersWithdraw(gomock.Any()).
		Return(float64(42), nil).
		AnyTimes()
	rm.
		EXPECT().
		CreateWithdrawal(gomock.Any(), models.Withdrawal{OrderNumber: "2377225624", Sum: 123}).
		Return(models.Withdrawal{OrderNumber: "2377225624", Sum: 123}, nil).
		AnyTimes()
	rm.
		EXPECT().
		GetUsersWithdrawals(gomock.Any()).
		Return([]models.Withdrawal{models.Withdrawal{OrderNumber: "2377225624", Sum: 123, ProcessedAt: processedAt}}, nil).
		AnyTimes()

	sh := NewRouter(options, &store)

	for _, tc := range testCases {
		t.Run(tc.method, func(t *testing.T) {

			r := httptest.NewRequest(tc.method, tc.path, nil)
			if tc.requestBody != "" {
				r = httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.requestBody))
			}
			w := httptest.NewRecorder()
			if tc.userID != "" {
				JWTToken, _ := auth.BuildJWTString(tc.userID)
				r.Header.Set("Cookie", "Authorization="+JWTToken)
			}
			r.Header.Set("Content-Type", tc.contentType)

			sh.ServeHTTP(w, r)

			assert.Equal(t, tc.expectedCode, w.Code, "Код ответа не совпадает с ожидаемым")
			if tc.expectedBody != "" {
				assert.Equal(t, tc.expectedBody, w.Body.String(), "Тело ответа не совпадает с ожидаемым")
			}
		})
	}
}
