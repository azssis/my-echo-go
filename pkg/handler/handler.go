package handler

import (
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

// Echo는 HTTP 요청을 에코하는 핸들러입니다.
func Echo(w http.ResponseWriter, r *http.Request) {
	// 1. 요청 데이터 수집
	reqData, err := collectRequestData(r)
	if err != nil {
		log.Printf("failed to collect request data: %s", err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// 2. 응답 생성
	msg, err := formatResponse(reqData)
	if err != nil {
		log.Printf("failed to format response: %s", err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// 3. 응답 전송
	if _, err := w.Write(msg); err != nil {
		log.Printf("failed to write response: %s", err.Error())
	}
}

// RequestData는 HTTP 요청의 모든 관련 정보를 담는 구조체입니다.
type RequestData struct {
	Header any // map[string]any 또는 string
	Body   any
}

// collectRequestData는 HTTP 요청으로부터 헤더와 본문 데이터를 수집합니다.
func collectRequestData(r *http.Request) (*RequestData, error) {
	// 1. 헤더 정보 수집
	headerInfo := summarizeRequestHeader(r)

	// 2. 본문 읽기
	defer r.Body.Close()
	bodyData, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body: %w", err)
	}

	// 3. 본문 파싱
	parsedBody, err := parseRequestBody(bodyData, r.Header.Get("Content-Type"))
	if err != nil {
		// 파싱 실패 시 원본 문자열 사용
		parsedBody = string(bodyData)
	}

	return &RequestData{
		Header: headerInfo,
		Body:   parsedBody,
	}, nil
}

// parseRequestBody는 Content-Type에 따라 요청 본문을 적절한 형식으로 파싱합니다.
func parseRequestBody(data []byte, contentType string) (any, error) {
	switch {
	case strings.HasPrefix(contentType, "text/"),
		strings.HasPrefix(contentType, "application/xml"):
		return string(data), nil
	case strings.HasPrefix(contentType, "application/json"):
		var result any
		if err := json.Unmarshal(data, &result); err != nil {
			return nil, fmt.Errorf("failed to parse json: %w", err)
		}
		return result, nil
	case strings.HasPrefix(contentType, "application/yaml"):
		var result any
		if err := yaml.Unmarshal(data, &result); err != nil {
			return nil, fmt.Errorf("failed to parse yaml: %w", err)
		}
		return result, nil
	default:
		return string(data), nil
	}
}

// summarizeRequestHeader summarizes key details from an http.Request into a map including headers and cookies.
func summarizeRequestHeader(r *http.Request) any {
	if option.GlobalOption.HeaderMode == option.HeaderSimple {
		// 간단한 헤더 정보를 하나의 문자열로 포맷팅
		return fmt.Sprintf("%s %s %s [%s]",
			r.Method,
			r.RequestURI,
			r.Host,
			time.Now().Format(time.StampMicro),
		)
	}

	// 상세 헤더 정보 포함
	req := map[string]any{
		"Time":          time.Now().Format(time.StampMicro),
		"Remote":        r.RemoteAddr,
		"Host":          r.Host,
		"Method":        r.Method,
		"Proto":         r.Proto,
		"URI":           r.RequestURI,
		"ContentLength": r.ContentLength,
		"UserAgent":     r.UserAgent(),
		"Headers":       extractHeaders(r),
	}

	if len(r.TransferEncoding) > 0 {
		req["Encoding"] = r.TransferEncoding
	}
	if cookies := extractCookies(r); cookies != nil {
		req["Cookies"] = cookies
	}

	return req
}

// Extract headers into a map
func extractHeaders(r *http.Request) map[string]interface{} {
	headers := make(map[string]interface{})
	for k, v := range r.Header {
		headers[k] = strings.Join(v, ",")
	}
	return headers
}

// Extract cookies as a slice
func extractCookies(r *http.Request) []*http.Cookie {
	if len(r.Cookies()) > 0 {
		return r.Cookies()
	}
	return nil
}

// formatResponse는 수집된 요청 데이터를 지정된 형식으로 변환합니다.
func formatResponse(data *RequestData) ([]byte, error) {
	response := make(map[string]any)

	// 옵션에 따라 헤더와 바디 정보를 선택적으로 포함
	if option.GlobalOption.ShowHeader {
		response["request"] = data.Header
	}

	if option.GlobalOption.ShowBody {
		// 바디가 문자열이고 비어있지 않은 경우에만 포함
		if str, ok := data.Body.(string); ok && str == "" {
			// 빈 바디는 제외
		} else {
			response["body"] = data.Body
		}
	}

	// 아무 정보도 표시하지 않을 경우 기본 메시지 포함
	if len(response) == 0 {
		response["message"] = "Request received but no content is configured to be shown"
	}

	return formatOutput(response)
}

// formatOutput serializes a value into the configured format with size limits and pretty printing.
func formatOutput(v interface{}) ([]byte, error) {
	const EllipsisSize = 3

	// Helper function: serializes JSON with or without indentation
	serializeJSON := func(value interface{}, pretty bool) ([]byte, error) {
		if pretty {
			return json.MarshalIndent(value, "", "  ")
		}
		return json.Marshal(value)
	}

	// Helper function: performs string cleaning
	cleanString := func(input string) string {
		result := strings.ReplaceAll(input, "\\n", "")
		result = strings.ReplaceAll(result, "\\", "")
		result = strings.ReplaceAll(result, "\"", "")
		return result
	}

	var serialized []byte
	var err error

	switch option.GlobalOption.Format {
	case option.FormatJSON:
		serialized, err = serializeJSON(v, option.GlobalOption.Pretty)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize to JSON: %w", err)
		}
	case option.FormatYAML:
		serialized, err = yaml.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize to YAML: %w", err)
		}
	default: // TEXT
		switch val := v.(type) {
		case string:
			serialized = []byte(val)
		case []byte:
			serialized = val
		default:
			// For other types, convert to JSON and clean it
			jsonBytes, err := serializeJSON(v, option.GlobalOption.Pretty)
			if err != nil {
				return nil, fmt.Errorf("failed to serialize to text: %w", err)
			}
			serialized = []byte(cleanString(string(jsonBytes)))
		}
	}

	// Adjust size by limiting the output and adding ellipsis if necessary
	if int64(len(serialized)) > option.GlobalOption.Size {
		adjustedSize := option.GlobalOption.Size + EllipsisSize
		copy(serialized[option.GlobalOption.Size:], []byte("..."))
		serialized = serialized[:adjustedSize]
	}

	return serialized, nil
}
