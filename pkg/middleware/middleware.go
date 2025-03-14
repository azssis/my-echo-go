package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type Option struct {
	Quiet          bool   // whether to include request content into response
	Size           int64  // max size of payload
	Pretty         bool   // whether to pretty print json
	Format         string // json or text
	DetailedHeader bool   // whether to show detailed header information
}

// responseWriter는 응답을 캡처하기 위한 http.ResponseWriter의 래퍼입니다.
type responseWriter struct {
	http.ResponseWriter
	body       bytes.Buffer
	statusCode int
}

// Write는 응답 본문을 캡처하고 원래의 ResponseWriter에도 씁니다.
func (w *responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// WriteHeader는 상태 코드를 캡처하고 원래의 ResponseWriter에도 씁니다.
func (w *responseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

var option Option
var hostname string = "localhost"

const (
	DefaultLen = 1024
	MaxLen     = DefaultLen * 1024 * 4
)

func init() {
	option = Option{
		Quiet:          true,
		Size:           MaxLen,
		Pretty:         false,
		Format:         "TEXT",
		DetailedHeader: false,
	}
	var err error
	hostname, err = os.Hostname()
	if err != nil {
		hostname = "localhost"
	}
}

// StringToBool converts a string to a boolean value
func StringToBool(str string) bool {
	switch strings.ToLower(str) {
	case "true", "1", "yes":
		return true
	case "false", "0", "no", "":
		return false
	default:
		return false
	}
}

func InitOption() {
	option.Pretty = StringToBool(os.Getenv("LOG_PRETTY"))
	option.DetailedHeader = StringToBool(os.Getenv("LOG_DETAILED_HEADER"))

	switch strings.ToUpper(os.Getenv("LOG_LEVEL")) {
	case "FATAL", "ERROR", "WARN":
		// 기본값 유지
	case "INFO":
		option.Quiet = false
		option.Size = DefaultLen
	case "DEBUG":
		option.Quiet = false
		option.Size = DefaultLen * 10
	case "TRACE":
		option.Quiet = false
		option.Size = MaxLen
	default:
		option.Quiet = true
		option.Size = DefaultLen
	}
}

// formatLogMessage는 로그 메시지를 JSON 또는 텍스트 형식으로 포맷팅합니다.
func formatLogMessage(r *http.Request, rw *responseWriter, duration time.Duration) string {
	logData := map[string]interface{}{
		"timestamp": time.Now().Format(time.StampMicro),
		"duration":  duration.String(),
		"status":    rw.statusCode,
		"method":    r.Method,
		"uri":       r.RequestURI,
		"host":      r.Host,
	}

	if option.DetailedHeader {
		logData["remote_addr"] = r.RemoteAddr
		logData["proto"] = r.Proto
		logData["user_agent"] = r.UserAgent()
		logData["content_length"] = r.ContentLength
		if len(r.TransferEncoding) > 0 {
			logData["transfer_encoding"] = r.TransferEncoding
		}
	}

	if !option.Quiet {
		if rw.body.Len() > 0 {
			respBody := rw.body.String()
			if len(respBody) > int(option.Size) {
				respBody = respBody[:option.Size] + "..."
			}
			logData["response"] = respBody
		}
	}

	if option.Pretty {
		jsonBytes, err := json.MarshalIndent(logData, "", "  ")
		if err == nil {
			return string(jsonBytes)
		}
	}

	jsonBytes, err := json.Marshal(logData)
	if err != nil {
		return fmt.Sprintf("Error formatting log message: %v", err)
	}
	return string(jsonBytes)
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// 응답 캡처를 위한 래퍼 생성
		rw := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// 핸들러 실행
		next.ServeHTTP(rw, r)

		// 로그 출력
		duration := time.Since(start)
		log.Println(formatLogMessage(r, rw, duration))
	})
}
