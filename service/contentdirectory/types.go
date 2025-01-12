package contentdirectory

import (
	"encoding/xml"
	"fmt"
	"upnp-mediaserver/epgstation"
	"log"
	"path/filepath"
	"strconv"
	"time"
)

type ObjectID string

var JST = time.FixedZone("Asia/Tokyo", 9*60*60)
var registory = make(map[ObjectID]interface{})
var resRegistory = make(map[ObjectID]interface{})

type Container struct {
	XMLName xml.Name `xml:"urn:schemas-upnp-org:metadata-1-0/DIDL-Lite/ container"`

	Id         ObjectID `xml:"urn:schemas-upnp-org:metadata-1-0/DIDL-Lite/ id,attr"`
	ParentID   ObjectID `xml:"urn:schemas-upnp-org:metadata-1-0/DIDL-Lite/ parentID,attr"`
	Title      string   `xml:"http://purl.org/dc/elements/1.1/ title"`
	Class      string   `xml:"urn:schemas-upnp-org:metadata-1-0/upnp/ class"`
	Restricted string   `xml:"urn:schemas-upnp-org:metadata-1-0/DIDL-Lite/ restricted,attr"`

	ChildCount int           `xml:"urn:schemas-upnp-org:metadata-1-0/DIDL-Lite/ childCount,attr"`
	Children   []interface{} `xml:"-"`
}

func (c *Container) AppendContainer(child *Container) {
	c.Children = append(c.Children, child)
	c.ChildCount++
	registory[child.Id] = child
}

func (c *Container) AppendItem(item *Item) {
	c.Children = append(c.Children, item)
	c.ChildCount++
	registory[item.Id] = item
}

type Item struct {
	XMLName xml.Name `xml:"urn:schemas-upnp-org:metadata-1-0/DIDL-Lite/ item"`

	Id         ObjectID `xml:"urn:schemas-upnp-org:metadata-1-0/DIDL-Lite/ id,attr"`
	ParentID   ObjectID `xml:"urn:schemas-upnp-org:metadata-1-0/DIDL-Lite/ parentID,attr"`
	Title      string   `xml:"http://purl.org/dc/elements/1.1/ title"`
	Class      string   `xml:"urn:schemas-upnp-org:metadata-1-0/upnp/ class"`
	Restricted string   `xml:"urn:schemas-upnp-org:metadata-1-0/DIDL-Lite/ restricted,attr"`

	Date      string `xml:"http://purl.org/dc/elements/1.1/ date"`
	Resources *[]Res

	AlbumArtURI *string `xml:"urn:schemas-upnp-org:metadata-1-0/upnp/ albumArtURI"`
}

type Res struct {
	XMLName      xml.Name      `xml:"urn:schemas-upnp-org:metadata-1-0/DIDL-Lite/ res"`
	ProtocolInfo string        `xml:"urn:schemas-upnp-org:metadata-1-0/DIDL-Lite/ protocolInfo,attr"`
	Size         int           `xml:"urn:schemas-upnp-org:metadata-1-0/DIDL-Lite/ size,attr"`
	Duration     string        `xml:"urn:schemas-upnp-org:metadata-1-0/DIDL-Lite/ duration,attr"`
	DurationNS   time.Duration `xml:"-"`
	URL          string        `xml:",chardata"`
}

// <DIDL-Lite xmlns="urn:schemas-upnp-org:metadata-1-0/DIDL-Lite/" xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:upnp="urn:schemas-upnp-org:metadata-1-0/upnp/">

type DIDLLite struct {
	XMLName xml.Name `xml:"urn:schemas-upnp-org:metadata-1-0/DIDL-Lite/ DIDL-Lite"`
	Objects []interface{}
}

func NewContainer(Id ObjectID, Parent *Container, Title string) *Container {
	var objectID, parentID ObjectID
	switch Parent {
	case nil:
		parentID, objectID = ObjectID("-1"), ObjectID("0")
	default:
		parentID, objectID = Parent.Id, Id
	}
	container := &Container{
		Id:         objectID,
		ParentID:   parentID,
		Title:      Title,
		Class:      "object.container",
		Restricted: "true",
		Children:   make([]interface{}, 0),
		ChildCount: 0,
	}
	registory[container.Id] = container
	if Parent != nil {
		Parent.AppendContainer(container)
	}
	return container
}

func fmtProtocolInfo(videoFile *epgstation.VideoFile) (string, error) {
	var mime, pn, op, ci string

	switch filepath.Ext(*videoFile.Filename) {
	case ".m2ts":
		mime = "video/mpeg"
		pn = "MPEG_PS_NTSC"
		op = "10"
		ci = "0"
	case ".mp4":
		mime = "video/mp4"
		pn = "AVC_MP4_BL_CIF15_AAC_520"
		op = "01"
		ci = "1"
	case ".mkv":
		mime = "video/x-matroska"
		pn = "AVC_MKV_HP_HD_AAC_MULT5"
		op = "01"
		ci = "1"
	default:
		return "", fmt.Errorf("unknown filetype %s", filepath.Ext(*videoFile.Filename))
	}
	return fmt.Sprintf("http-get:*:%s:DLNA_ORG.PN=%s;DLNA.ORG_OP=%s;DLNA.ORG_CI=%s;DLNA.ORG_FLAGS=01118000000000000000000000000000", mime, pn, op, ci), nil
}

func fmtDuration(d time.Duration) string {
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	d -= s * time.Second
	ms := d / time.Millisecond

	return fmt.Sprintf("%d:%02d:%02d.%03d", h, m, s, ms)
}

func NewResource(videoFile *epgstation.VideoFile, duration time.Duration) Res {
	protocolInfo, err := fmtProtocolInfo(videoFile)
	if err != nil {
		log.Fatal(err)
	}
	res := Res{
		ProtocolInfo: protocolInfo,
		URL:          fmt.Sprintf("%svideos/recorded?videoFileId=%d", serviceURLBase, videoFile.Id),
		Size:         videoFile.Size,
		Duration:     fmtDuration(duration),
		DurationNS:   duration,
	}
	objectId := strconv.Itoa(int(videoFile.Id))
	resRegistory[ObjectID(objectId)] = &res
	return res
}

func NewItem(Parent *Container, recordedItem *epgstation.RecordedItem, videoFileIdDurationMap map[epgstation.VideoFileId]time.Duration) *Item {
	if Parent == nil {
		log.Fatal("container is required for item")
	}

	resources := make([]Res, len(*recordedItem.VideoFiles))
	for i, videoFile := range *recordedItem.VideoFiles {
		// Some videoFile may deleted from filesystem manually. In such case, mapping entry not found 
		if duration, ok := videoFileIdDurationMap[videoFile.Id]; ok {
			resources[i] = NewResource(&videoFile, duration)
		}
	}
	item := &Item{
		Id:         ObjectID(strconv.Itoa(int(recordedItem.Id))),
		ParentID:   Parent.Id,
		Title:      recordedItem.Name,
		Class:      "object.item.videoItem",
		Restricted: "true",

		Resources: &resources,

		Date: time.Unix(int64(recordedItem.StartAt)/1000, 0).In(JST).Format("2006-01-02"),
	}
	if len(*recordedItem.Thumbnails) > 0 {
		albumArtURI := fmt.Sprintf("%s/thumbnails/%d", epgstation.ServerAPIRoot, (*recordedItem.Thumbnails)[0])
		item.AlbumArtURI = &albumArtURI
	}
	Parent.AppendItem(item)
	return item
}
