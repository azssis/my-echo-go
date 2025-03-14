package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"my-echo-go/pkg/option"
	"net/http"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// responseWriter는 응답을 캡처하기 위한 http.ResponseWriter의 래퍼입니다.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader는 상태 코드를 캡처하고 원래의 ResponseWriter에도 씁니다.
func (w *responseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

// formatLogMessage는 로그 메시지를 지정된 형식으로 포맷팅합니다.
func formatLogMessage(r *http.Request, reqBody []byte, rw *responseWriter, duration time.Duration) string {
	// 기본 메타데이터
	baseMetadata := map[string]interface{}{
		"timestamp": time.Now().Format("2006-01-02 15:04:05.000000"),
		"duration":  fmt.Sprintf("%.3fms", float64(duration.Microseconds())/1000.0),
		"status":    fmt.Sprintf("%d %s", rw.statusCode, http.StatusText(rw.statusCode)),
		"method":    r.Method,
		"uri":       r.RequestURI,
		"host":      r.Host,
	}

	// 헤더 정보 수집
	headers := make(map[string]interface{})
	// 기본 헤더 정보
	simpleHeaders := map[string]interface{}{
		"remote_addr": r.RemoteAddr,
		"user_agent":  r.UserAgent(),
	}
	// 표준 헤더 정보
	standardHeaders := map[string]interface{}{
		"remote_addr":    r.RemoteAddr,
		"proto":          r.Proto,
		"user_agent":     r.UserAgent(),
		"content_length": r.ContentLength,
	}
	// 상세 헤더 정보
	detailedHeaders := map[string]interface{}{
		"remote_addr":    r.RemoteAddr,
		"proto":          r.Proto,
		"user_agent":     r.UserAgent(),
		"content_length": r.ContentLength,
		"host":           r.Host,
		"method":         r.Method,
		"request_uri":    r.RequestURI,
		"content_type":   r.Header.Get("Content-Type"),
		"accept":         r.Header.Get("Accept"),
	}
	if len(r.TransferEncoding) > 0 {
		detailedHeaders["transfer_encoding"] = r.TransferEncoding
	}

	// 헤더 모드에 따라 헤더 선택
	switch option.GlobalOption.HeaderMode {
	case option.HeaderSimple:
		headers = simpleHeaders
	case option.HeaderStandard:
		headers = standardHeaders
	case option.HeaderDetailed:
		headers = detailedHeaders
	}

	// 로그 데이터 구성
	logData := baseMetadata

	// 헤더와 본문 정보 추가
	if option.GlobalOption.ShowHeader {
		logData["headers"] = headers
	}

	// 요청 본문 추가
	if option.GlobalOption.ShowBody && len(reqBody) > 0 {
		reqBodyStr := string(reqBody)
		if len(reqBodyStr) > int(option.GlobalOption.Size) {
			reqBodyStr = reqBodyStr[:option.GlobalOption.Size] + "..."
		}
		logData["payload"] = reqBodyStr
	}

	var result string
	var err error

	switch option.GlobalOption.Format {
	case option.FormatJSON:
		if option.GlobalOption.Pretty {
			jsonBytes, err := json.MarshalIndent(logData, "", "    ")
			if err == nil {
				return string(jsonBytes)
			}
		}
		jsonBytes, err := json.Marshal(logData)
		if err == nil {
			result = string(jsonBytes)
		}
	case option.FormatYAML:
		yamlBytes, err := yaml.Marshal(logData)
		if err == nil {
			result = string(yamlBytes)
		}
	default: // TEXT
		var builder strings.Builder

		// 헤더 모드에 따라 다른 형식으로 출력
		switch option.GlobalOption.HeaderMode {
		case option.HeaderSimple:
			// 기본 정보와 헤더를 한 줄에 표시
			builder.WriteString(fmt.Sprintf("[%s] %s %s %s %s",
				baseMetadata["timestamp"],
				baseMetadata["method"],
				baseMetadata["uri"],
				baseMetadata["status"],
				baseMetadata["duration"],
			))
			if option.GlobalOption.ShowHeader {
				builder.WriteString(fmt.Sprintf(" (from: %s, agent: %s)",
					headers["remote_addr"],
					headers["user_agent"],
				))
			}
			// 페이로드가 있는 경우 한 줄에 추가
			if option.GlobalOption.ShowBody && len(reqBody) > 0 {
				builder.WriteString(fmt.Sprintf(" payload: %v", logData["payload"]))
			}

		case option.HeaderStandard:
			// 기본 정보는 첫 줄에 표시
			builder.WriteString(fmt.Sprintf("[%s] %s %s %s %s",
				baseMetadata["timestamp"],
				baseMetadata["method"],
				baseMetadata["uri"],
				baseMetadata["status"],
				baseMetadata["duration"],
			))

			// 헤더와 페이로드는 별도 줄에 표시
			if option.GlobalOption.ShowHeader || (option.GlobalOption.ShowBody && len(reqBody) > 0) {
				builder.WriteString("\n")
				if option.GlobalOption.ShowHeader {
					builder.WriteString("├ Headers ─────────────────\n")
					for k, v := range headers {
						builder.WriteString(fmt.Sprintf("│ %-12s: %v\n", k, v))
					}
				}
				if option.GlobalOption.ShowBody && len(reqBody) > 0 {
					builder.WriteString("├ Payload ─────────────────\n")
					builder.WriteString(fmt.Sprintf("│ %v\n", logData["payload"]))
				}
				builder.WriteString("└──────────────")
			}

		case option.HeaderDetailed:
			// 모든 정보를 구조화된 형식으로 표시
			builder.WriteString("┌ Request Info ──────────────\n")
			builder.WriteString(fmt.Sprintf("│ %-12s: %s\n", "Timestamp", baseMetadata["timestamp"]))
			builder.WriteString(fmt.Sprintf("│ %-12s: %s\n", "Duration", baseMetadata["duration"]))
			builder.WriteString(fmt.Sprintf("│ %-12s: %s\n", "Status", baseMetadata["status"]))
			builder.WriteString(fmt.Sprintf("│ %-12s: %s\n", "Method", baseMetadata["method"]))
			builder.WriteString(fmt.Sprintf("│ %-12s: %s\n", "URI", baseMetadata["uri"]))
			builder.WriteString(fmt.Sprintf("│ %-12s: %s\n", "Host", baseMetadata["host"]))

			if option.GlobalOption.ShowHeader {
				builder.WriteString("├ Headers ─────────────────\n")
				for k, v := range headers {
					builder.WriteString(fmt.Sprintf("│ %-12s: %v\n", k, v))
				}
			}
			if option.GlobalOption.ShowBody && len(reqBody) > 0 {
				builder.WriteString("├ Payload ─────────────────\n")
				builder.WriteString(fmt.Sprintf("│ %v\n", logData["payload"]))
			}
			builder.WriteString("└─────────────────────")
		}

		result = builder.String()
	}

	if err != nil {
		return fmt.Sprintf("Error formatting log message: %v", err)
	}
	return result
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// 요청 본문 읽기
		var reqBody []byte
		if r.Body != nil {
			var err error
			reqBody, err = io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "Error reading request body", http.StatusInternalServerError)
				return
			}
			// 읽은 본문을 다시 Body에 설정
			r.Body = io.NopCloser(bytes.NewBuffer(reqBody))
		}

		// 응답 캡처를 위한 래퍼 생성
		rw := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// 핸들러 실행
		next.ServeHTTP(rw, r)

		// 로그 출력
		duration := time.Since(start)
		log.Println(formatLogMessage(r, reqBody, rw, duration))
	})
}
