package protocol

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
)

const (
	internalServerErrorMsg = "internal server error"
	incorrectRequestErrMsg = "incorrect request"
)

type Logger interface {
	Error(msg string, err error)
}

type Validator interface {
	Struct(s any) error
}

type ResponseHandler struct {
	logger    Logger
	validator Validator
}

func NewResponseHandler(logger Logger, opts ...Option) *ResponseHandler {
	h := &ResponseHandler{logger: logger}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

type ExceptionResponse struct {
	ErrorMessage string    `json:"errorMessage"`
	Status       int       `json:"status"`
	Timestamp    time.Time `json:"timestamp"`
}

func (h *ResponseHandler) SendErrorResponse(w http.ResponseWriter, status int, err error) {
	defaultMsg := ""
	switch status {
	case http.StatusInternalServerError:
		defaultMsg = internalServerErrorMsg
	case http.StatusBadRequest:
		defaultMsg = incorrectRequestErrMsg
	}

	msg := strings.TrimSpace(fmt.Sprintf("%s %s", defaultMsg, flattenError(err)))

	body, marshalErr := json.Marshal(&ExceptionResponse{
		ErrorMessage: msg,
		Status:       status,
		Timestamp:    time.Now().Truncate(time.Second),
	})
	if marshalErr != nil {
		h.logger.Error("SendErrorResponse: marshal", marshalErr)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(marshalErr.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, writeErr := w.Write(body)
	if writeErr != nil {
		h.logger.Error("SendErrorResponse: write", writeErr)
	}
}

func (h *ResponseHandler) SendSuccessResponse(w http.ResponseWriter, status int, body any) {
	payload, err := json.Marshal(body)
	if err != nil {
		h.logger.Error("SendSuccessResponse: marshal", err)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, writeErr := w.Write(payload)
	if writeErr != nil {
		h.logger.Error("SendSuccessResponse: write", writeErr)
	}
}

// BindJSON закрывает body после чтения, в том числе при ошибке.
func (h *ResponseHandler) BindJSON(body io.ReadCloser, params any) error {
	defer func() { _ = body.Close() }()

	if err := json.NewDecoder(body).Decode(params); err != nil {
		return fmt.Errorf("decode body: %w", err)
	}
	if h.validator == nil {
		return nil
	}
	err := h.validator.Struct(params)
	if err != nil {
		return err
	}
	return nil
}

func (h *ResponseHandler) Validate(params any) error {
	if h.validator == nil {
		return nil
	}
	return h.validator.Struct(params)
}

func flattenError(err error) string {
	if err == nil {
		return ""
	}
	var verrs validator.ValidationErrors
	if !errors.As(err, &verrs) {
		return err.Error()
	}
	parts := make([]string, 0, len(verrs))
	for _, fe := range verrs {
		parts = append(parts, formatFieldError(fe))
	}
	return strings.Join(parts, "; ")
}

func formatFieldError(fe validator.FieldError) string {
	field := fe.Field()
	switch fe.Tag() {
	case "required":
		return field + " is required"
	case "min":
		return fmt.Sprintf("%s must be at least %s", field, fe.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s", field, fe.Param())
	case "gte":
		return fmt.Sprintf("%s must be >= %s", field, fe.Param())
	case "lte":
		return fmt.Sprintf("%s must be <= %s", field, fe.Param())
	default:
		return fmt.Sprintf("%s fails %q", field, fe.Tag())
	}
}
