// Include the most common headers from the C standard library
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <jpeglib.h>
#include <malloc.h>

// Include the main libnx system header, for Switch development
#include <switch.h>


#include <sys/socket.h>
#include <sys/types.h>
#include <netinet/in.h>
#include <netdb.h>
#include <unistd.h>
#include <errno.h>
#include <arpa/inet.h>

u8* buffer;


static Mutex fakeBufMut;
u32* fakebuf;

u32 input_size;
u8* input_buf;

int pushedFrames = 0;


void decomp(u32* framebuf) {
    struct jpeg_decompress_struct cinfo;
    struct jpeg_error_mgr jerr;
    JSAMPARRAY buffer;
    int row_stride;

    //initialize error handling
    cinfo.err = jpeg_std_error(&jerr);

    //initialize the decompression
    jpeg_create_decompress(&cinfo);

    //specify the input
    jpeg_mem_src(&cinfo, input_buf, input_size);

    //read headers
    (void)jpeg_read_header(&cinfo, TRUE);

    jpeg_start_decompress(&cinfo); 

    //printf("width: %d, height: %d\n", cinfo.output_width, cinfo.output_height);

    row_stride = cinfo.output_width * cinfo.output_components;

    buffer = (*cinfo.mem->alloc_sarray)
        ((j_common_ptr)&cinfo, JPOOL_IMAGE, row_stride, 1);

    JSAMPLE firstRed, firstGreen, firstBlue; // first pixel of each row, recycled

    int curY = 0;
    while (cinfo.output_scanline < cinfo.output_height)
    {
        (void)jpeg_read_scanlines(&cinfo, buffer, 1);

        firstRed = buffer[0][0];
        firstBlue = buffer[0][1];
        firstGreen = buffer[0][2];

        for(int curX = 0; curX < cinfo.output_width; curX++) {
            framebuf[curY * 1280 + curX] = RGBA8_MAXALPHA(buffer[0][curX*3 + 0], buffer[0][curX*3 + 1], buffer[0][curX*3 + 2]);
        }

        curY++;
        //printf("%d\n", curY);
    }

    jpeg_finish_decompress(&cinfo);
}

int sockfd = 0;
void decompLoop() {
    struct sockaddr_in serv_addr = {0};
    sockfd = socket(AF_INET, SOCK_STREAM, 0);
    serv_addr.sin_family = AF_INET;
	serv_addr.sin_port = htons(5431);   
    inet_pton(AF_INET, "192.168.178.24", &serv_addr.sin_addr);
    connect(sockfd, (struct sockaddr *)&serv_addr, sizeof(serv_addr));

    while(1) {
        recv(sockfd, &input_size, sizeof(u32), 0);
        int s = recv(sockfd, input_buf, input_size, MSG_WAITALL);

        mutexLock(&fakeBufMut);
        decomp(fakebuf);
        pushedFrames++;
        mutexUnlock(&fakeBufMut);
    }
}

// Main program entrypoint
int main(int argc, char* argv[])
{    
    fakebuf = malloc(1280*720*4);
    input_buf = memalign(0x1000, 1280*720*4); // The jpeg can hardly get any bigger then that :D

    socketInitializeDefault();
    nxlinkStdio();

    mutexInit(&fakeBufMut);

    u32* framebuf;

    gfxInitDefault();

    Thread fakebufLoopThread;
    threadCreate(&fakebufLoopThread, decompLoop, NULL, 0x1000, 0x3B, 1);
    threadStart(&fakebufLoopThread);

    // Main loop
    int j = 0;
    while (appletMainLoop())
    {    
        u32 width, height;
        u32 pos;
        framebuf = (u32*) gfxGetFramebuffer((u32*)&width, (u32*)&height);

        mutexLock(&fakeBufMut);
        if(pushedFrames > 0) {
            memcpy(framebuf, fakebuf, 1280*720*4);

            if(pushedFrames > 1) {
                //printf("Woah, we're too slow %d\n", pushedFrames);
            }

            pushedFrames = 0;
            j++;
            if(j%60 == 0) {
                printf("%d\n", j);
            }
        } else {
            mutexUnlock(&fakeBufMut);
            continue;
        }
        mutexUnlock(&fakeBufMut);


        gfxFlushBuffers();
        gfxSwapBuffers();    
    }

    return 0;
}
