package mp4

import (
	"github.com/AlexxIT/go2rtc/cmd/api"
	"github.com/AlexxIT/go2rtc/cmd/streams"
	"github.com/AlexxIT/go2rtc/pkg/mp4"
	"github.com/AlexxIT/go2rtc/pkg/streamer"
	"strings"
)

const packetSize = 8192

func handlerWS(tr *api.Transport, msg *api.Message) {
	src := tr.Request.URL.Query().Get("src")
	stream := streams.GetOrNew(src)
	if stream == nil {
		return
	}

	cons := &mp4.Consumer{}
	cons.UserAgent = tr.Request.UserAgent()
	cons.RemoteAddr = tr.Request.RemoteAddr

	if codecs, ok := msg.Value.(string); ok {
		cons.Medias = parseMedias(codecs, true)
	}

	cons.Listen(func(msg interface{}) {
		if data, ok := msg.([]byte); ok {
			for len(data) > packetSize {
				tr.Write(data[:packetSize])
				data = data[packetSize:]
			}
			tr.Write(data)
		}
	})

	if err := stream.AddConsumer(cons); err != nil {
		log.Warn().Err(err).Caller().Send()
		tr.Error(err)
		return
	}

	tr.OnClose(func() {
		stream.RemoveConsumer(cons)
	})

	tr.Write(&api.Message{Type: "mse", Value: cons.MimeType()})

	data, err := cons.Init()
	if err != nil {
		log.Warn().Err(err).Caller().Send()
		tr.Error(err)
		return
	}

	tr.Write(data)

	cons.Start()
}

func handlerWS4(tr *api.Transport, msg *api.Message) {
	src := tr.Request.URL.Query().Get("src")
	stream := streams.GetOrNew(src)
	if stream == nil {
		return
	}

	cons := &mp4.Segment{}

	if codecs, ok := msg.Value.(string); ok {
		cons.Medias = parseMedias(codecs, false)
	}

	cons.Listen(func(msg interface{}) {
		if data, ok := msg.([]byte); ok {
			tr.Write(data)
		}
	})

	if err := stream.AddConsumer(cons); err != nil {
		log.Error().Err(err).Caller().Send()
		return
	}

	tr.OnClose(func() {
		stream.RemoveConsumer(cons)
	})
}

func parseMedias(codecs string, parseAudio bool) (medias []*streamer.Media) {
	var videos []*streamer.Codec
	var audios []*streamer.Codec

	for _, name := range strings.Split(codecs, ",") {
		switch name {
		case "avc1.640029":
			codec := &streamer.Codec{Name: streamer.CodecH264}
			videos = append(videos, codec)
		case "hvc1.1.6.L153.B0":
			codec := &streamer.Codec{Name: streamer.CodecH265}
			videos = append(videos, codec)
		case "mp4a.40.2":
			codec := &streamer.Codec{Name: streamer.CodecAAC}
			audios = append(audios, codec)
		}
	}

	if videos != nil {
		media := &streamer.Media{
			Kind:      streamer.KindVideo,
			Direction: streamer.DirectionRecvonly,
			Codecs:    videos,
		}
		medias = append(medias, media)
	}

	if audios != nil && parseAudio {
		media := &streamer.Media{
			Kind:      streamer.KindAudio,
			Direction: streamer.DirectionRecvonly,
			Codecs:    audios,
		}
		medias = append(medias, media)
	}

	return
}