package protocol

type Option func(h *ResponseHandler)

func WithValidation(v Validator) Option {
	return func(h *ResponseHandler) {
		h.validator = v
	}
}
