package oiio

/*
#include "stdlib.h"

#include "cpp/oiio.h"

*/
import "C"

import (
	"errors"
	"runtime"
	"unsafe"
)

// ImageInput abstracts the reading of an image file in a file format-agnostic manner.
type ImageInput struct {
	ptr unsafe.Pointer
}

func newImageInput(i unsafe.Pointer) *ImageInput {
	in := &ImageInput{i}
	runtime.SetFinalizer(in, deleteImageInput)
	return in
}

func deleteImageInput(i *ImageInput) {
	if i.ptr != nil {
		C.ImageInput_close(i.ptr)
		C.free(i.ptr)
		i.ptr = nil
	}
	runtime.KeepAlive(i)
}

// Create an ImageInput subclass instance that is able to read the given file and open it,
// returning the opened ImageInput if successful. If it fails, return error.
func OpenImageInput(filename string) (*ImageInput, error) {
	c_str := C.CString(filename)
	defer C.free(unsafe.Pointer(c_str))

	cfg := unsafe.Pointer(nil)
	ptr := C.ImageInput_Open(c_str, cfg)

	in := newImageInput(ptr)

	return in, in.LastError()
}

// Return the last error generated by API calls.
// An nil error will be returned if no error has occured.
func (i *ImageInput) LastError() error {
	c_str := C.ImageInput_geterror(i.ptr)
	runtime.KeepAlive(i)
	if c_str == nil {
		return nil
	}
	err := C.GoString(c_str)
	if err == "" {
		return nil
	}
	return errors.New(err)
}

// Open file with given name. Return true if the file was found and opened okay.
func (i *ImageInput) Open(filename string) error {
	i.Close()

	deleteImageInput(i)

	c_str := C.CString(filename)
	defer C.free(unsafe.Pointer(c_str))

	cfg := unsafe.Pointer(nil)
	ptr := C.ImageInput_Open(c_str, cfg)
	i.ptr = ptr

	return i.LastError()
}

// Close an image that we are totally done with.
func (i *ImageInput) Close() error {
	if !bool(C.ImageInput_close(i.ptr)) {
		return i.LastError()
	}
	return nil
}

// Return the name of the format implemented by this image.
func (i *ImageInput) FormatName() string {
	ret := C.GoString(C.ImageInput_format_name(i.ptr))
	runtime.KeepAlive(i)
	return ret
}

// Return true if the named file is file of the type for this ImageInput.
// The implementation will try to determine this as efficiently as possible,
// in most cases much less expensively than doing a full Open().
// Note that a file can appear to be of the right type (i.e., ValidFIle() returning true)
// but still fail a subsequent call to Open(), such as if the contents of the file are
// truncated, nonsensical, or otherwise corrupted.
func (i *ImageInput) ValidFile(filename string) bool {
	c_str := C.CString(filename)
	defer C.free(unsafe.Pointer(c_str))
	ret := bool(C.ImageInput_valid_file(i.ptr, c_str))
	runtime.KeepAlive(i)
	return ret
}

// Given the name of a 'feature', return whether this ImageOutput
// supports output of images with the given properties.
// Feature names that ImageIO plugins are expected to recognize
// include:
//    "tiles"          Is this format able to write tiled images?
//    "rectangles"     Does this plugin accept arbitrary rectangular
//                       pixel regions, not necessarily aligned to
//                       scanlines or tiles?
//    "random_access"  May tiles or scanlines be written in
//                       any order (false indicates that they MUST
//                       be in successive order).
//    "multiimage"     Does this format support multiple subimages
//                       within a file?
//    "appendsubimage" Does this format support adding subimages one at
//                       a time through open(name,spec,AppendSubimage)?
//                       If not, then open(name,subimages,specs) must
//                       be used instead.
//    "mipmap"         Does this format support multiple resolutions
//                       for an image/subimage?
//    "volumes"        Does this format support "3D" pixel arrays?
//    "rewrite"        May the same scanline or tile be sent more than
//                       once?  (Generally, this will be true for
//                       plugins that implement interactive display.)
//    "empty"          Does this plugin support passing a NULL data
//                       pointer to write_scanline or write_tile to
//                       indicate that the entire data block is zero?
//    "channelformats" Does the plugin/format support per-channel
//                       data formats?
//    "displaywindow"  Does the format support display ("full") windows
//                        distinct from the pixel data window?
//    "origin"         Does the format support a nonzero x,y,z
//                        origin of the pixel data window?
//    "negativeorigin" Does the format support negative x,y,z
//                        and full_{x,y,z} origin values?
//    "deepdata"       Deep (multi-sample per pixel) data
//
// Note that main advantage of this approach, versus having
// separate individual supports_foo() methods, is that this allows
// future expansion of the set of possible queries without changing
// the API, adding new entry points, or breaking linkage
// compatibility.
func (i *ImageInput) Supports(feature string) bool {
	c_str := C.CString(feature)
	defer C.free(unsafe.Pointer(c_str))
	ret := bool(C.ImageInput_supports(i.ptr, c_str))
	runtime.KeepAlive(i)
	return ret
}

// Return a reference to the image format specification of the current subimage/MIPlevel.
// Note that the contents of the spec are invalid before Open() or after Close(), and may
// change with a call to SeekSubImage().
func (i *ImageInput) Spec() *ImageSpec {
	ptr := C.ImageInput_spec(i.ptr)
	runtime.KeepAlive(i)
	return &ImageSpec{ptr}
}

// CurrentSubimage returns the index of the subimage that is currently being read.
// The first subimage (or the only subimage, if there is just one) is number 0.
func (i *ImageInput) CurrentSubimage() int {
	ret := int(C.ImageInput_current_subimage(i.ptr))
	runtime.KeepAlive(i)
	return ret
}

// Seek to the given subimage within the open image file.
// The first subimage of the file has index 0. Return true on
// success, false on failure (including that there is not a
// subimage with the specified index). The new
// subimage's vital statistics are put in newspec (and also saved
// in ImageInput.Spec()). The reader is expected to give the appearance
// of random access to subimages -- in other words,
// if it can't randomly seek to the given subimage, it should
// transparently close, reopen, and sequentially read through prior
// subimages.
func (i *ImageInput) SeekSubimage(index int, newSpec *ImageSpec) bool {
	if index < 0 {
		index = 0
	}

	if newSpec == nil || newSpec.ptr == nil {
		newSpec = NewImageSpec(TypeUnknown)
	}

	ok := C.ImageInput_seek_subimage(i.ptr, C.int(index), newSpec.ptr)
	runtime.KeepAlive(i)
	runtime.KeepAlive(newSpec)
	return bool(ok)
}

// Returns the index of the MIPmap image that is currently being read.
// The highest-res MIP level (or the only level, if there is just
// one) is number 0.
func (i *ImageInput) CurrentMipLevel() int {
	ret := int(C.ImageInput_current_miplevel(i.ptr))
	runtime.KeepAlive(i)
	return ret
}

// Seek to the given subimage and MIP-map level within the open
// image file. The first subimage of the file has index 0, the
// highest-resolution MIP level has index 0. Return true on
// success, false on failure (including that there is not a
// subimage or MIP level with the specified index). The new
// subimage's vital statistics are put in newspec (and also saved
// in ImageInpit.Spec()). The reader is expected to give the appearance
// of random access to subimages and MIP levels -- in other words,
// if it can't randomly seek to the given subimage/level, it should
// transparently close, reopen, and sequentially read through prior
// subimages and levels.
func (i *ImageInput) SeekMipLevel(subimage, miplevel int, newSpec *ImageSpec) bool {
	if subimage < 0 {
		subimage = 0
	}
	if miplevel < 0 {
		miplevel = 0
	}

	if newSpec == nil || newSpec.ptr == nil {
		newSpec = NewImageSpec(TypeUnknown)
	}

	ok := C.ImageInput_seek_subimage_miplevel(i.ptr, C.int(subimage), C.int(miplevel), newSpec.ptr)
	runtime.KeepAlive(i)
	runtime.KeepAlive(newSpec)
	return bool(ok)
}

// Read the entire image of width * height * depth * channels into contiguous float32 pixels.
// Read tiles or scanlines automatically.
func (i *ImageInput) ReadImage() ([]float32, error) {
	spec := i.Spec()
	size := spec.Width() * spec.Height() * spec.Depth() * spec.NumChannels()
	pixels := make([]float32, size)
	pixels_ptr := (*C.float)(unsafe.Pointer(&pixels[0]))
	C.ImageInput_read_image_floats(i.ptr, pixels_ptr)

	return pixels, i.LastError()
}

// Read the entire image of width * height * depth * channels into contiguous pixels.
// Read tiles or scanlines automatically.
//
// This call supports passing a callback pointer to both track the progress,
// and to optionally abort the processing. The callback function will receive
// a float32 value indicating the percentage done of the processing, and should
// return true if the process should abort, and false if it should continue.
//
// The underlying type of data is determined by the given TypeDesc.
// Returned interface{} will be:
//     TypeUint8   => []uint8
//     TypeInt8    => []int8
//     TypeUint16  => []uint16
//     TypeInt16   => []int16
//     TypeUint    => []uint
//     TypeInt     => []int
//     TypeUint64  => []uint64
//     TypeInt64   => []int64
//     TypeHalf    => []float32
//     TypeFloat   => []float32
//     TypeDouble  => []float64
//
// Example:
//
//     // Without a callback
//     val, err := in.ReadImageFormat(TypeFloat, nil)
//     if err != nil {
//         panic(err.Error())
//     }
//     floatPixels := val.([]float32)
//
//     // With a callback
//     var cbk ProgressCallback = func(done float32) bool {
//         fmt.Printf("Progress: %0.2f\n", done)
//         // Keep processing (return true to abort)
//         return false
//     }
//     val, _ = in.ReadImageFormat(TypeFloat, &cbk)
//     floatPixels = val.([]float32)
//
func (i *ImageInput) ReadImageFormat(format TypeDesc, progress *ProgressCallback) (interface{}, error) {
	spec := i.Spec()

	pixel_iface, ptr, err := allocatePixelBuffer(spec, format)
	if err != nil {
		return nil, err
	}

	var cbk unsafe.Pointer = nil
	if progress != nil {
		cbk = unsafe.Pointer(progress)
	}

	C.ImageInput_read_image_format(i.ptr, (C.TypeDesc)(format), ptr, cbk)

	return pixel_iface, i.LastError()
}

// Read the scanline that includes pixels (*,y,z), converting if necessary
// from the native data format of the file into contiguous float32 pixels (z==0 for non-volume images).
// The size of the slice is: width * depth * channels
func (i *ImageInput) ReadScanline(y, z int) ([]float32, error) {
	spec := i.Spec()

	size := spec.Width() * spec.Depth() * spec.NumChannels()
	pixels := make([]float32, size)
	pixels_ptr := (*C.float)(unsafe.Pointer(&pixels[0]))
	C.ImageInput_read_scanline_floats(i.ptr, C.int(y), C.int(z), pixels_ptr)

	return pixels, i.LastError()
}

// Read the tile whose upper-left origin is (x,y,z),
// converting if necessary from the native data format of the file
// into contiguous float32 pixels.
// The size of the slice is: tilewidth * tileheight * depth * channels
// (z==0 for non-volume images.)
func (i *ImageInput) ReadTile(x, y, z int) ([]float32, error) {
	spec := i.Spec()

	size := spec.TilePixels()
	pixels := make([]float32, size)
	pixels_ptr := (*C.float)(unsafe.Pointer(&pixels[0]))
	C.ImageInput_read_tile_floats(i.ptr, C.int(x), C.int(y), C.int(z), pixels_ptr)

	return pixels, i.LastError()
}
