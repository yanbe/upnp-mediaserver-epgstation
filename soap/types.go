package soap

import (
	"encoding/xml"
	"upnp-mediaserver/service/contentdirectory"
)

type Request struct {
	XMLName       xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ Envelope"`
	EncodingStyle string   `xml:"http://schemas.xmlsoap.org/soap/envelope/ encodingStyle,attr"`
	Body          struct {
		XMLName               xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ Body"`
		Browse                *Browse
		GetSystemUpdateID     *GetSystemUpdateID
		GetSearchCapabilities *GetSearchCapabilities
		GetSortCapabilities   *GetSortCapabilities
	}
}

type Response struct {
	XMLName       xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ Envelope"`
	EncodingStyle string   `xml:"http://schemas.xmlsoap.org/soap/envelope/ encodingStyle,attr"`
	Body          struct {
		XMLName                       xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ Body"`
		BrowseResponse                *BrowseResponse
		GetSystemUpdateIDResponse     *GetSystemUpdateIDResponse
		GetSearchCapabilitiesResponse *GetSearchCapabilitiesResponse
		GetSortCapabilitiesResponse   *GetSortCapabilitiesResponse
	}
}

type Browse struct {
	XMLName        xml.Name `xml:"urn:schemas-upnp-org:service:ContentDirectory:1 Browse"`
	ObjectID       string
	BrowseFlag     string
	Filter         string
	StartingIndex  int
	RequestedCount int
	SortCriteria   string
}

type BrowseResponse struct {
	XMLName        xml.Name `xml:"urn:schemas-upnp-org:service:ContentDirectory:1 BrowseResponse"`
	Result         string
	NumberReturned int
	TotalMatches   int
	UpdateID       int
}

type GetSystemUpdateID struct {
	XMLName xml.Name `xml:"urn:schemas-upnp-org:service:ContentDirectory:1 GetSystemUpdateID"`
}

type GetSystemUpdateIDResponse struct {
	XMLName xml.Name `xml:"urn:schemas-upnp-org:service:ContentDirectory:1 GetSystemUpdateIDResponse"`
	Id      int
}

type GetSearchCapabilities struct {
	XMLName xml.Name `xml:"urn:schemas-upnp-org:service:ContentDirectory:1 GetSearchCapabilities"`
}

type GetSearchCapabilitiesResponse struct {
	XMLName    xml.Name `xml:"urn:schemas-upnp-org:service:ContentDirectory:1 GetSearchCapabilitiesResponse"`
	SearchCaps string
}

type GetSortCapabilities struct {
	XMLName xml.Name `xml:"urn:schemas-upnp-org:service:ContentDirectory:1 GetSortCapabilities"`
}

type GetSortCapabilitiesResponse struct {
	XMLName  xml.Name `xml:"urn:schemas-upnp-org:service:ContentDirectory:1 GetSortCapabilitiesResponse"`
	SortCaps string
}

// <DIDL-Lite xmlns="urn:schemas-upnp-org:metadata-1-0/DIDL-Lite/" xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:upnp="urn:schemas-upnp-org:metadata-1-0/upnp/">

type DIDLLite struct {
	XMLName    xml.Name `xml:"urn:schemas-upnp-org:metadata-1-0/DIDL-Lite/ DIDL-Lite"`
	Items      []*contentdirectory.Item
	Containers []*contentdirectory.Container
}
