package models

// ErrorDetails rappresenta la struttura dell'errore standard
type ErrorDetails struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// ErrorResponse è il wrapper per le risposte di errore
type ErrorResponse struct {
	Success bool         `json:"success"` // Sarà sempre false
	Error   ErrorDetails `json:"error"`
}

// Meta rappresenta i metadati opzionali per la paginazione
type Meta struct {
	Page  int `json:"page,omitempty"`
	Limit int `json:"limit,omitempty"`
	Total int `json:"total,omitempty"`
}

// SuccessResponse è il wrapper per le risposte positive
type SuccessResponse struct {
	Success bool        `json:"success"` // Sarà sempre true
	Data    interface{} `json:"data"`
	Meta    *Meta       `json:"meta,omitempty"`
}

// Helper per creare facilmente una risposta di errore
func NewErrorResponse(code, message string, details interface{}) ErrorResponse {
	return ErrorResponse{
		Success: false,
		Error: ErrorDetails{
			Code:    code,
			Message: message,
			Details: details,
		},
	}
}

// Helper per creare facilmente una risposta di successo
func NewSuccessResponse(data interface{}, meta *Meta) SuccessResponse {
	return SuccessResponse{
		Success: true,
		Data:    data,
		Meta:    meta,
	}
}
