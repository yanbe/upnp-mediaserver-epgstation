package contentdirectory

import (
	"context"
	"encoding/xml"
	"fmt"
	"upnp-mediaserver/epgstation"
	"log"
	"time"
)

var serviceURLBase string
var videoFileIdDurationMap map[epgstation.VideoFileId]time.Duration
var lastRecordedTotal int

var genreIdNameMap = map[epgstation.ProgramGenreLv1]string{
	0x0: "ニュース・報道",
	0x1: "スポーツ",
	0x2: "情報・ワイドショー",
	0x3: "ドラマ",
	0x4: "音楽",
	0x5: "バラエティ",
	0x6: "映画",
	0x7: "アニメ・特撮",
	0x8: "ドキュメンタリー・教養",
	0x9: "劇場・公演",
	0xa: "趣味・教育",
	0xb: "福祉",
	0xc: "予備",
	0xd: "予備",
	0xe: "拡張",
	0xf: "その他",
}

func watchEPGStationForSetup() {
	for {
		time.Sleep(1 * time.Minute)
		res, err := epgstation.EPGStation.GetRecordedWithResponse(context.Background(), &epgstation.GetRecordedParams{
			IsHalfWidth: false,
		})
		if err == nil && res.JSON200.Total != lastRecordedTotal {
			Setup(serviceURLBase)
		}
	}
}

func Setup(ServiceURLBase string) {
	log.Println("Setup ContentDirectory start")
	serviceURLBase = ServiceURLBase

	rootContainer := NewContainer("0", nil, "Root")
	log.Println("Setup Recorded Container")
	recordedContainer := setupRecordedContainer(rootContainer)
	log.Println("Setup Genres Container")
	setupGenresContainer(rootContainer)
	log.Println("Setup Channels Container")
	setupChannelsContainer(rootContainer)
	log.Println("Setup Rules Container")
	setupRulesContainer(rootContainer)

	log.Printf("Setup ContentDirectory complete. %d items found", recordedContainer.ChildCount)

	go watchEPGStationForSetup()
}

func setupRecordedContainer(parent *Container) *Container {
	recordedContainer := NewContainer("01", parent, "録画済み")
	res, err := epgstation.EPGStation.GetRecordedWithResponse(context.Background(), &epgstation.GetRecordedParams{
		IsHalfWidth: false,
	})
	if err != nil {
		log.Fatal(err)
	}
	lastRecordedTotal = res.JSON200.Total
	videoFileIdDurationMap = make(map[epgstation.VideoFileId]time.Duration)
	for _, recordedItem := range res.JSON200.Records {
		for _, videoFile := range *recordedItem.VideoFiles {
			res, err := epgstation.EPGStation.GetVideosVideoFileIdDurationWithResponse(context.Background(), epgstation.PathVideoFileId(videoFile.Id))
			if err != nil {
				log.Fatal(err)
			}
			if res.JSONDefault != nil {
				// Some videoFile may deleted from filesystem manually.  In such case, EPGstation returns error 
				log.Printf("Error (code: %d %s): %s", res.JSONDefault.Code, res.JSONDefault.Message, *res.JSONDefault.Errors)
				continue
			}
			videoFileIdDurationMap[videoFile.Id] = time.Duration(res.JSON200.Duration * float32(time.Second))
		}
	}
	for _, recordedItem := range res.JSON200.Records {
		NewItem(recordedContainer, &recordedItem, videoFileIdDurationMap)
	}
	return recordedContainer
}

func setupGenresContainer(parent *Container) *Container {
	genresContainer := NewContainer("02", parent, "ジャンル別")
	res, err := epgstation.EPGStation.GetRecordedOptionsWithResponse(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	for _, genre := range res.JSON200.Genres {
		genreContainer := NewContainer(ObjectID(fmt.Sprintf("02%d", int(genre.Genre))), genresContainer, genreIdNameMap[genre.Genre])
		genre := epgstation.QueryProgramGenre(genre.Genre)
		res, err := epgstation.EPGStation.GetRecordedWithResponse(context.Background(), &epgstation.GetRecordedParams{
			IsHalfWidth: true,
			Genre:       &genre,
		})
		if err != nil {
			log.Fatal(err)
		}
		for _, recordedItem := range res.JSON200.Records {
			NewItem(genreContainer, &recordedItem, videoFileIdDurationMap)
		}
	}
	return genresContainer
}

func setupChannelsContainer(parent *Container) *Container {
	channelsContainer := NewContainer("03", parent, "チャンネル別")
	resChannelInfo, err := epgstation.EPGStation.GetChannelsWithResponse(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	channelIdChannelItemMap := make(map[epgstation.ChannelId]epgstation.ChannelItem)
	for _, channelItem := range *resChannelInfo.JSON200 {
		channelIdChannelItemMap[channelItem.Id] = channelItem
	}

	res, err := epgstation.EPGStation.GetRecordedOptionsWithResponse(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	for _, channel := range res.JSON200.Channels {
		channelName := channelIdChannelItemMap[channel.ChannelId].HalfWidthName
		channelContainer := NewContainer(ObjectID(fmt.Sprintf("03%d", int(channel.ChannelId))), channelsContainer, channelName)
		queryChannelId := epgstation.QueryChannelId(channel.ChannelId)
		res, err := epgstation.EPGStation.GetRecordedWithResponse(context.Background(), &epgstation.GetRecordedParams{
			IsHalfWidth: false,
			ChannelId:   &queryChannelId,
		})
		if err != nil {
			log.Fatal(err)
		}
		for _, recordedItem := range res.JSON200.Records {
			NewItem(channelContainer, &recordedItem, videoFileIdDurationMap)
		}
	}
	return channelsContainer
}

func setupRulesContainer(parent *Container) *Container {
	rulesContainer := NewContainer("04", parent, "ルール別")
	resRulesInfo, err := epgstation.EPGStation.GetRulesKeywordWithResponse(context.Background(), &epgstation.GetRulesKeywordParams{})
	if err != nil {
		log.Fatal(err)
	}
	for _, ruleItem := range resRulesInfo.JSON200.Items {
		queryRuleId := epgstation.QueryRuleId(ruleItem.Id)
		res, err := epgstation.EPGStation.GetRecordedWithResponse(context.Background(), &epgstation.GetRecordedParams{
			IsHalfWidth: false,
			RuleId:      &queryRuleId,
		})
		if err != nil {
			log.Fatal(err)
		}
		if res.JSON200.Total > 0 {
			ruleContainer := NewContainer(ObjectID(fmt.Sprintf("04%d", int(ruleItem.Id))), rulesContainer, ruleItem.Keyword)
			for _, recordedItem := range res.JSON200.Records {
				NewItem(ruleContainer, &recordedItem, videoFileIdDurationMap)
			}
		}
	}
	return rulesContainer
}

func GetRecordedTotal() int {
	res, err := epgstation.EPGStation.GetRecordedWithResponse(context.Background(), &epgstation.GetRecordedParams{
		IsHalfWidth: false,
	})
	if err != nil {
		log.Fatal(err)
	}
	return res.JSON200.Total
}

func MarshalMetadata(objectID string) string {
	object := registory[ObjectID(objectID)]
	wrapper := DIDLLite{}
	wrapper.Objects = append(wrapper.Objects, &object)
	data, err := xml.Marshal(wrapper)
	if err != nil {
		log.Fatal(err)
	}
	return string(data)
}

func MarshalDirectChildren(objectID string, StartingIndex int, RequestedCount int) string {
	object := registory[ObjectID(objectID)]
	container, ok := object.(*Container)
	if !ok {
		log.Fatalf("passed objectID %s not found as a container", objectID)
	}
	wrapper := DIDLLite{}
	var min, max int
	if StartingIndex < cap(container.Children) {
		min = StartingIndex
	} else {
		min = cap(container.Children)
	}
	if StartingIndex+RequestedCount <= cap(container.Children) {
		max = StartingIndex + RequestedCount
	} else {
		max = cap(container.Children)
	}
	wrapper.Objects = container.Children[min:max]
	data, err := xml.Marshal(wrapper)
	if err != nil {
		log.Fatal(err)
	}
	return string(data)
}

func GetObject(objectID string) interface{} {
	return registory[ObjectID(objectID)]
}
