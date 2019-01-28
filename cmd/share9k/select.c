#if defined(_WIN32) || defined(__APPLE__)

#ifdef _WIN32
#include "Windows.h"
static HWND hwnd          = 0;
#endif

#include <stdio.h>
#include <sys/time.h>
#include <SDL2/SDL.h>
#include <SDL2/SDL_syswm.h>

static SDL_Window *screen = NULL;
static SDL_Renderer *rend = NULL;
static int desktop_height = 0;
static int desktop_width = 0;

#ifdef _WIN32
void acquire_full_desktop_info(
  int *x,
  int *y,
  int *w,
  int *h
) {
  *x = GetSystemMetrics(SM_XVIRTUALSCREEN);
  *y = GetSystemMetrics(SM_YVIRTUALSCREEN);
  *w = GetSystemMetrics(SM_CXVIRTUALSCREEN);
  *h = GetSystemMetrics(SM_CYVIRTUALSCREEN);
}
#elif __APPLE__
extern void getScreenResMac(int *x, int *y, int *w, int *h);

void acquire_full_desktop_info(
  int *x,
  int *y,
  int *w,
  int *h
) {
  getScreenResMac(x, y, w, h);
}
#endif

void acquire_rectangle(
  int aspect_ratio_x,
  int aspect_ratio_y,
  int *x,
  int *y,
  int *w,
  int *h
) {
  int desktop_height = 0;
  int desktop_width = 0;
  int desktop_origin_x = 0;
  int desktop_origin_y = 0;

  acquire_full_desktop_info(
    &desktop_origin_x,
    &desktop_origin_y,
    &desktop_width,
    &desktop_height
  );

  SDL_Init(SDL_INIT_VIDEO);

  screen = SDL_CreateWindow(
    "Share9k Select",
    desktop_origin_x,
    desktop_origin_y,
    desktop_width,
    desktop_height,
    SDL_SWSURFACE | SDL_WINDOW_BORDERLESS | SDL_WINDOW_ALWAYS_ON_TOP
  );

#ifdef _WIN32
  SDL_SysWMinfo info;
  SDL_VERSION(&info.version);
  if(SDL_GetWindowWMInfo(screen, &info))
  {
    hwnd = info.info.win.window;
  }
  SetWindowLong(hwnd, GWL_EXSTYLE, GetWindowLong(hwnd, GWL_EXSTYLE) |WS_EX_LAYERED);
  SetLayeredWindowAttributes(hwnd, RGB(0,0,0), (255 * 70)/100, LWA_COLORKEY);
#endif

  rend = SDL_CreateRenderer(screen, -1, SDL_RENDERER_ACCELERATED);

  SDL_SetWindowOpacity(screen, 0.5f);

  int start_x = 0;
  int start_y = 0;

  int cur_x = 0;
  int cur_y = 0;

  int clicked = 0;

  SDL_Event evt;

  for (; ;)
  {
    SDL_Rect box;
    
    while(SDL_PollEvent(&evt))
    {
      if (evt.type == SDL_MOUSEBUTTONDOWN)
      {
        clicked = 1;
        start_x = evt.button.x;
        start_y = evt.button.y;
        cur_x = start_x;
        cur_y = start_y;
      }

      if (evt.type == SDL_MOUSEBUTTONUP)
      {
        clicked = 2;
        cur_x = evt.button.x;
        cur_y = evt.button.y;
      }

      if (evt.type == SDL_MOUSEMOTION)
      {
        cur_x = evt.motion.x;
        cur_y = evt.motion.y;
      }
    }

    SDL_RenderClear(rend);

    if (clicked)
    {
      if (start_x > cur_x)
      {
        box.x = cur_x;
        box.w = start_x - cur_x;
      }
      else
      {
        box.x = start_x;
        box.w = cur_x - start_x;
      }

      if (start_y > cur_y)
      {
        box.y = cur_y;
        box.h = start_y - cur_y;
      }
      else
      {
        box.y = start_y;
        box.h = cur_y - start_y;
      }

      // Aspect ratio restriction enabled?
      if (aspect_ratio_x > 0)
      {
        float ar_x = aspect_ratio_x;
        float ar_y = aspect_ratio_y;
        float ar = ar_y / ar_x;

        float actual_width = box.w;
        float actual_height = box.h;

        float new_height = actual_width * ar;

        box.h = new_height;
      }

      SDL_SetRenderDrawColor(rend, 0, 0, 255, 0);
      SDL_RenderDrawRect(rend, &box);
    }

    SDL_SetRenderDrawColor(rend, 0, 0, 0, 0);
    // SDL_RenderCopy(rend, text, NULL, &textr);
    SDL_RenderPresent(rend);

    if (clicked == 2)
    {
      SDL_DestroyRenderer(rend);
      SDL_DestroyWindow(screen);

      // Account for difference beteween SDL window and actual desktop origin
      *x = box.x + desktop_origin_x;
      *y = box.y + desktop_origin_y;
      *w = box.w;
      *h = box.h;
      return;
    }
  }
}

#else

#include <X11/Xlib.h>
#include <X11/Xutil.h>
#include <X11/XKBlib.h>
#include <X11/extensions/XInput2.h>

void acquire_rectangle(
  int *x,
  int *y,
  int *w,
  int *h,
) {
  Display *display = XOpenDisplay(":0");
  if (display == NULL)
  {
    printf("Could not acquire dislay.\n");
    exit(0);
  }

  Window root = XDefaultRootWindow(display);
  XGrabPointer(display, root, False,
               ButtonPressMask |
                 ButtonReleaseMask |
                 ButtonMotionMask,
               GrabModeAsync,
               GrabModeAsync,
               RootWindow(display, DefaultScreen(display)),
               None,
               CurrentTime);

  int button_pressed = 0;
  int rx = 0, ry = 0;
  int rect_x = 0, rect_y = 0, rect_w = 0, rect_h = 0;
  GC gc;
  XGCValues gcval;

  gcval.foreground = XWhitePixel(display, 0);
  gcval.function = GXxor;
  gcval.background = XBlackPixel(display, 0);
  gcval.plane_mask = gcval.background ^ gcval.foreground;
  gcval.subwindow_mode = IncludeInferiors;
  
  gc = XCreateGC(display, root,
              GCFunction | GCForeground | GCBackground | GCSubwindowMode,
              &gcval);
  XEvent evt;

  int scanning = 1;
  while(scanning)
  {
    XNextEvent(display, &evt);
    printf("got event\n");
    switch(evt.type)
    {
      case MotionNotify:
      printf("notifying motion\n");
      if (button_pressed) {
        if (rect_w) {
          /* re-draw the last rect to clear it */
          XDrawRectangle(display, root, gc, rect_x, rect_y, rect_w, rect_h);
        }

        rect_x = rx;
        rect_y = ry;
        rect_w = evt.xmotion.x - rect_x;
        rect_h = evt.xmotion.y - rect_y;

        if (rect_w < 0) {
          rect_x += rect_w;
          rect_w = 0 - rect_w;
        }
        if (rect_h < 0) {
          rect_y += rect_h;
          rect_h = 0 - rect_h;
        }
        /* draw rectangle */
        XDrawRectangle(display, root, gc, rect_x, rect_y, rect_w, rect_h);
        XFlush(display);
      }
      break;

      case ButtonPress:
      printf("pressed button\n");
      button_pressed = 1;
      rx = evt.xbutton.x;
      ry = evt.xbutton.y;
      break;

      case ButtonRelease:
      printf("released button\n");
      goto end;
      break;
    }
  }
end:
  if (rect_w) {
    XDrawRectangle(display, root, gc, rect_x, rect_y, rect_w, rect_h);
    XFlush(display);
  }

  XFreeGC(display, gc);
  XSync(display, True);
  XCloseDisplay(display);

  *x = rect_x;
  *y = rect_y;
  *w = rect_w;
  *h = rect_h;
}

#endif
