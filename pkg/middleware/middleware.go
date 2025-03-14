package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"my-echo-go/pkg/util"
	"net/http"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Option struct {
	Quiet          bool   // whether to include request content into response
	Size           int64  // max size of payload
	Pretty         bool   // whether to pretty print json
	Format         string // json or text
	DetailedHeader bool   // whether to show detailed header information
	ShowBody       bool   // whether to show response body in log
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
		ShowBody:       false,
	}
	var err error
	hostname, err = os.Hostname()
	if err != nil {
		hostname = "localhost"
	}
}

func InitOption() {
	switch strings.ToUpper(os.Getenv("LOG_LEVEL")) {
	case "FATAL", "ERROR", "WARN":
		// 기본값 유지
	case "INFO":
		option.Quiet = false
		option.Size = DefaultLen
		option.DetailedHeader = false
		option.ShowBody = false
	case "DEBUG":
		option.Quiet = false
		option.Size = DefaultLen * 10
		option.DetailedHeader = true
		option.ShowBody = true
	case "TRACE":
		option.Quiet = false
		option.Size = MaxLen
		option.DetailedHeader = true
		option.ShowBody = true
	default:
		option.Quiet = true
		option.Size = DefaultLen
		option.ShowBody = false
	}

	if os.Getenv("LOG_PRETTY") != "" {
		option.Pretty = util.StringToBool(os.Getenv("LOG_PRETTY"))
	}
	if os.Getenv("LOG_DETAILED_HEADER") != "" {
		option.DetailedHeader = util.StringToBool(os.Getenv("LOG_DETAILED_HEADER"))
	}
	if os.Getenv("LOG_FORMAT") != "" {
		option.Format = strings.ToUpper(os.Getenv("LOG_FORMAT"))
	}
	if os.Getenv("LOG_SHOW_BODY") != "" {
		option.ShowBody = util.StringToBool(os.Getenv("LOG_SHOW_BODY"))
	}

	// 최종 적용된 옵션을 구조화하여 출력
	var logLevel string
	switch {
	case option.Quiet:
		logLevel = "FATAL/ERROR/WARN"
	case option.Size == DefaultLen:
		logLevel = "INFO"
	case option.Size == DefaultLen*10:
		logLevel = "DEBUG"
	case option.Size == MaxLen:
		logLevel = "TRACE"
	default:
		logLevel = "UNKNOWN"
	}

	log.Printf(`
┌ Logging Options
│ Level          : %s
│ Format         : %s
│ Max Size       : %d bytes
│ Pretty Print   : %t
│ Show Headers   : %t
│ Show Body      : %t
│ Host          : %s
└──────────────`,
		logLevel,
		option.Format,
		option.Size,
		option.Pretty,
		option.DetailedHeader,
		option.ShowBody,
		hostname,
	)
}

// formatLogMessage는 로그 메시지를 지정된 형식으로 포맷팅합니다.
func formatLogMessage(r *http.Request, rw *responseWriter, duration time.Duration) string {
	logData := make(map[string]interface{})

	// 기본 메타데이터
	baseMetadata := map[string]interface{}{
		"timestamp": time.Now().Format("2006-01-02 15:04:05.000000"),
		"duration":  fmt.Sprintf("%.3fms", float64(duration.Microseconds())/1000.0),
		"status":    fmt.Sprintf("%d %s", rw.statusCode, http.StatusText(rw.statusCode)),
		"method":    r.Method,
		"uri":       r.RequestURI,
		"host":      r.Host,
	}

	if option.DetailedHeader {
		// DetailedHeader가 true일 때는 기본 메타데이터를 그대로 사용
		logData = baseMetadata
		// 추가 헤더 정보를 구조화하여 저장
		headers := make(map[string]interface{})
		headers["remote_addr"] = r.RemoteAddr
		headers["proto"] = r.Proto
		headers["user_agent"] = r.UserAgent()
		headers["content_length"] = r.ContentLength
		if len(r.TransferEncoding) > 0 {
			headers["transfer_encoding"] = r.TransferEncoding
		}
		logData["headers"] = headers
	} else {
		// DetailedHeader가 false일 때는 메타데이터를 한 줄로 직렬화
		metadataBytes, _ := json.Marshal(baseMetadata)
		logData["meta"] = json.RawMessage(metadataBytes)
	}

	// 응답 본문이 있고 quiet 모드가 아니고 ShowBody가 true인 경우 별도 필드로 추가
	if !option.Quiet && option.ShowBody && rw.body.Len() > 0 {
		respBody := rw.body.String()
		if len(respBody) > int(option.Size) {
			respBody = respBody[:option.Size] + "..."
		}
		// 응답이 JSON 형식인 경우 구조화하여 저장
		var bodyData interface{}
		if err := json.Unmarshal([]byte(respBody), &bodyData); err == nil {
			logData["body"] = bodyData
		} else {
			logData["body"] = respBody
		}
	}

	var result string
	var err error

	switch option.Format {
	case "JSON":
		if option.Pretty {
			jsonBytes, err := json.MarshalIndent(logData, "", "    ")
			if err == nil {
				return string(jsonBytes)
			}
		}
		jsonBytes, err := json.Marshal(logData)
		if err == nil {
			result = string(jsonBytes)
		}
	case "YAML":
		yamlBytes, err := yaml.Marshal(logData)
		if err == nil {
			result = string(yamlBytes)
		}
	default: // TEXT
		var builder strings.Builder
		if !option.DetailedHeader {
			// 메타데이터를 구조화된 형식으로 표시
			builder.WriteString("┌ Request Info\n")
			metaData := logData["meta"].(json.RawMessage)
			var meta map[string]interface{}
			json.Unmarshal(metaData, &meta)
			for k, v := range meta {
				builder.WriteString(fmt.Sprintf("│ %-12s: %v\n", k, v))
			}
			builder.WriteString("└──────────────\n")
		} else {
			// 상세 정보를 구조화된 형식으로 표시
			builder.WriteString("┌ Request Info\n")
			for k, v := range logData {
				if k != "headers" && k != "body" {
					builder.WriteString(fmt.Sprintf("│ %-12s: %v\n", k, v))
				}
			}
			builder.WriteString("├ Headers\n")
			headers := logData["headers"].(map[string]interface{})
			for k, v := range headers {
				builder.WriteString(fmt.Sprintf("│ %-12s: %v\n", k, v))
			}
			builder.WriteString("└──────────────\n")
		}

		// 응답 본문이 있는 경우 추가
		if body, ok := logData["body"]; ok {
			builder.WriteString("┌ Response Body\n")
			switch v := body.(type) {
			case map[string]interface{}, []interface{}:
				// JSON 데이터인 경우 들여쓰기 적용
				prettyJSON, _ := json.MarshalIndent(v, "│ ", "  ")
				builder.WriteString(fmt.Sprintf("│ %s\n", string(prettyJSON)))
			default:
				// 일반 텍스트인 경우
				builder.WriteString(fmt.Sprintf("│ %v\n", body))
			}
			builder.WriteString("└──────────────")
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
