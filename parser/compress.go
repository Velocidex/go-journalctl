package parser

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"sync"

	"github.com/klauspost/compress/zstd"
	"github.com/pierrec/lz4/v4"
	"github.com/ulikunitz/xz"
)

// Compression flag bits in ObjectHeader.flags(); see OBJECT_COMPRESSED_* in
// https://github.com/systemd/systemd/blob/main/src/libsystemd/sd-journal/journal-def.h
const (
	objectCompressedXZ   = 0x01
	objectCompressedLZ4  = 0x02
	objectCompressedZSTD = 0x04
)

var (
	zstd_decoder      *zstd.Decoder
	zstd_decoder_once sync.Once
)

func getZSTDDecoder() (*zstd.Decoder, error) {
	var err error
	zstd_decoder_once.Do(func() {
		zstd_decoder, err = zstd.NewReader(nil, zstd.WithDecoderConcurrency(0))
	})
	return zstd_decoder, err
}

// ZSTD: payload is a raw ZSTD frame (starts with magic 0xFD2FB528)
func decompressZSTD(compressed []byte) ([]byte, error) {
	if len(compressed) < 4 {
		return nil, fmt.Errorf("zstd: payload too short (%d bytes)", len(compressed))
	}
	dec, err := getZSTDDecoder()
	if err != nil {
		return nil, fmt.Errorf("zstd: failed to init decoder: %w", err)
	}
	out, err := dec.DecodeAll(compressed, nil)
	if err != nil {
		return nil, fmt.Errorf("zstd: failed to decompress: %w", err)
	}
	return out, nil
}

// LZ4 (older journals): payload is a raw LZ4 block prefixed by a
// little-endian uint64 giving the original (decompressed) size.
func decompressLZ4(compressed []byte) ([]byte, error) {
	if len(compressed) < 8 {
		return nil, fmt.Errorf("lz4: payload too short (%d bytes)", len(compressed))
	}
	orig_size := int(binary.LittleEndian.Uint64(compressed[:8]))
	dst := make([]byte, orig_size)
	n, err := lz4.UncompressBlock(compressed[8:], dst)
	if err != nil {
		return nil, fmt.Errorf("lz4: failed to decompress: %w", err)
	}
	return dst[:n], nil
}

// XZ (oldest journals, systemd < 216): payload is a raw XZ stream.
func decompressXZ(compressed []byte) ([]byte, error) {
	r, err := xz.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return nil, fmt.Errorf("xz: failed to init reader: %w", err)
	}
	out, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("xz: failed to decompress: %w", err)
	}
	return out, nil
}

// decompressPayload decompresses a DATA object payload if the object flags
// indicate compression, then returns the raw bytes.
func decompressPayload(flags byte, compressed []byte) ([]byte, error) {
	switch {
	case flags&objectCompressedXZ != 0:
		return decompressXZ(compressed)
	case flags&objectCompressedZSTD != 0:
		return decompressZSTD(compressed)
	case flags&objectCompressedLZ4 != 0:
		return decompressLZ4(compressed)
	default:
		return nil, fmt.Errorf("unknown compression flags: 0x%02x", flags)
	}
}
