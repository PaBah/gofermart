package server

import (
	"compress/flate"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/PaBah/gofermart/internal/auth"
	"github.com/PaBah/gofermart/internal/config"
	"github.com/PaBah/gofermart/internal/dto"
	"github.com/PaBah/gofermart/internal/logger"
	"github.com/PaBah/gofermart/internal/models"
	"github.com/PaBah/gofermart/internal/storage"
	"github.com/PaBah/gofermart/internal/utils"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

type Server struct {
	options *config.Options
	storage storage.Repository
}

func (s Server) registerUserHandle(res http.ResponseWriter, req *http.Request) {
	contentType := req.Header.Get("Content-Type")
	if contentType != "application/json" {
		http.Error(res, "Invalid request content type", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	requestData := &dto.RegisterUserRequest{}
	err = json.Unmarshal(body, requestData)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}
	user := models.NewUser(requestData.Login, requestData.Password)
	createdUser, err := s.storage.CreateUser(req.Context(), user)

	if errors.Is(err, storage.ErrAlreadyExists) {
		http.Error(res, "User with such login already exists", http.StatusConflict)
		return
	}

	JWTToken, err := auth.BuildJWTString(createdUser.ID)
	if err != nil {
		http.Error(res, "Can not build auth token", http.StatusInternalServerError)
		return
	}
	res.Header().Set("Content-Type", "application/json")
	http.SetCookie(res, &http.Cookie{Name: "Authorization", Value: JWTToken})
}

func (s Server) loginUserHandle(res http.ResponseWriter, req *http.Request) {
	contentType := req.Header.Get("Content-Type")
	if contentType != "application/json" {
		http.Error(res, "Invalid request content type", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	requestData := &dto.LoginUserRequest{}
	err = json.Unmarshal(body, requestData)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}
	user, err := s.storage.AuthorizeUser(req.Context(), requestData.Login)

	if err != nil || !utils.CheckPasswordHash(user.Password, requestData.Password) {
		http.Error(res, "User with such credentials can not be logined", http.StatusUnauthorized)
		return
	}

	JWTToken, err := auth.BuildJWTString(user.ID)
	if err != nil {
		http.Error(res, "Can not build auth token", http.StatusInternalServerError)
		return
	}
	res.Header().Set("Content-Type", "application/json")
	http.SetCookie(res, &http.Cookie{Name: "Authorization", Value: JWTToken})
}

func (s Server) getOrdersHandle(res http.ResponseWriter, req *http.Request) {
	orders, err := s.storage.GetUsersOrders(req.Context())
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(orders) == 0 {
		res.WriteHeader(http.StatusNoContent)
		return
	}

	var responseData []dto.ActualOrderStateResponse
	for _, order := range orders {
		responseData = append(responseData, dto.ActualOrderStateResponse{
			Number:     order.Number,
			Status:     order.Status,
			Accrual:    order.Accrual,
			UploadedAt: dto.JSONTime(order.UploadedAt),
		})
	}

	res.Header().Set("Content-Type", "application/json")
	response, err := json.Marshal(responseData)

	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	res.WriteHeader(http.StatusOK)
	_, err = res.Write(response)
	if err != nil {
		logger.Log().Error("Can not send response from GET /api/user/orders", zap.Error(err))
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s Server) createOrderHandle(res http.ResponseWriter, req *http.Request) {
	contentType := req.Header.Get("Content-Type")
	if contentType != "text/plain" {
		http.Error(res, "Invalid request content type", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	orderNumber := string(body)
	err = utils.ValidateLuhn(orderNumber)
	if err != nil {
		http.Error(res, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	order, err := s.storage.RegisterOrder(req.Context(), string(body))
	if err != nil {
		if order.UserID == req.Context().Value(auth.ContextUserKey).(string) {
			res.WriteHeader(http.StatusOK)
			return
		}

		res.WriteHeader(http.StatusConflict)
		return
	}
	res.WriteHeader(http.StatusAccepted)
}

func (s Server) getBalanceHandle(res http.ResponseWriter, req *http.Request) {
	balance, err := s.storage.GetUsersBalance(req.Context())
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	withdraw, err := s.storage.GetUsersWithdraw(req.Context())
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	responseData := dto.UserBalanceResponse{
		Current:   balance - withdraw,
		Withdrawn: withdraw,
	}

	res.Header().Set("Content-Type", "application/json")
	response, err := json.Marshal(responseData)

	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	res.WriteHeader(http.StatusOK)
	_, err = res.Write(response)
	if err != nil {
		logger.Log().Error("Can not send response from GET /api/user/balance", zap.Error(err))
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s Server) withdrawFundsHandle(res http.ResponseWriter, req *http.Request) {
	contentType := req.Header.Get("Content-Type")
	if contentType != "application/json" {
		http.Error(res, "Invalid request content type", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	requestData := &dto.WithdrawalRequest{}
	err = json.Unmarshal(body, requestData)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	err = utils.ValidateLuhn(requestData.Number)
	if err != nil {
		http.Error(res, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	balance, err := s.storage.GetUsersBalance(req.Context())
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	withdraw, err := s.storage.GetUsersWithdraw(req.Context())
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	if balance-withdraw < requestData.Sum {
		http.Error(res, "Not enough funds", http.StatusPaymentRequired)
		return
	}

	withdrawal := models.Withdrawal{OrderNumber: requestData.Number, Sum: requestData.Sum}
	_, err = s.storage.CreateWithdrawal(req.Context(), withdrawal)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}
}

func (s Server) getUsersWithdrawalsHandle(res http.ResponseWriter, req *http.Request) {
	withdrawals, err := s.storage.GetUsersWithdrawals(req.Context())
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(withdrawals) == 0 {
		res.WriteHeader(http.StatusNoContent)
		return
	}

	var responseData []dto.WithdrawalsResponse
	for _, withdrawal := range withdrawals {
		responseData = append(responseData, dto.WithdrawalsResponse{
			OrderNumber: withdrawal.OrderNumber,
			Sum:         withdrawal.Sum,
			ProcessedAt: dto.JSONTime(withdrawal.ProcessedAt),
		})
	}

	res.Header().Set("Content-Type", "application/json")
	response, err := json.Marshal(responseData)

	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	res.WriteHeader(http.StatusOK)
	_, err = res.Write(response)
	if err != nil {
		logger.Log().Error("Can not send response from GET /api/user/withdrawals", zap.Error(err))
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
}

func NewRouter(options *config.Options, storage *storage.Repository) *chi.Mux {
	r := chi.NewRouter()

	s := Server{
		options: options,
		storage: *storage,
	}
	r.Use(logger.LoggerMiddleware)
	//r.Use(middleware.Logger)
	r.Use(middleware.NewCompressor(flate.DefaultCompression).Handler)

	r.Group(func(r chi.Router) {
		r.Post("/api/user/register", s.registerUserHandle)
		r.Post("/api/user/login", s.loginUserHandle)
	})
	r.Group(func(r chi.Router) {
		r.Use(auth.AuthorizedMiddleware)
		r.Post("/api/user/orders", s.createOrderHandle)
		r.Get("/api/user/orders", s.getOrdersHandle)
		r.Get("/api/user/balance", s.getBalanceHandle)
		r.Post("/api/user/balance/withdraw", s.withdrawFundsHandle)
		r.Get("/api/user/withdrawals", s.getUsersWithdrawalsHandle)
	})
	r.MethodNotAllowed(
		func(writer http.ResponseWriter, request *http.Request) {
			writer.WriteHeader(http.StatusBadRequest)
		},
	)
	return r
}
