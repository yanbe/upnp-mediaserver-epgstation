package soap

import (
	"encoding/xml"
	"io/ioutil"
	"net/http"
	"reflect"
	"regexp"
	"log"
)

const actionNameRegexp = `"urn:schemas-upnp-org:service:ContentDirectory:1#(.+)"`

func HandleAction(r *http.Request) []byte {
	actionName := regexp.MustCompile(actionNameRegexp).FindStringSubmatch(r.Header.Get("SoapAction"))[1]
	log.Printf("Handling action: %s", actionName);
	data, _ := ioutil.ReadAll(r.Body)
	var soapReq Request
	xml.Unmarshal(data, &soapReq)
	reqStruct := reflect.ValueOf(soapReq.Body).FieldByName(actionName).Elem()
	argv := make([]reflect.Value, reqStruct.NumField()-1)
	for i := range argv {
		argv[i] = reqStruct.Field(i + 1) // skip XMLName field
	}
	result := reflect.ValueOf(&Action{}).MethodByName(actionName).Call(argv)

	var soapRes Response
	soapRes.EncodingStyle = "http://schemas.xmlsoap.org/soap/encoding/"
	resStructFieldPtr := reflect.ValueOf(soapRes.Body).FieldByName(actionName + "Response")
	resStructPtr := reflect.New(resStructFieldPtr.Type().Elem())
	for i, v := range result {
		resStructPtr.Elem().Field(i + 1).Set(v) // skip XMLName field
	}
	reflect.ValueOf(&soapRes.Body).Elem().FieldByName(actionName + "Response").Set(resStructPtr)
	res, _ := xml.Marshal(soapRes)
	return res
}
