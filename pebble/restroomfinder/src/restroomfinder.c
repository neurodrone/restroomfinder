#include <pebble.h>

#define COLOR_WINDOW_BG GColorWhite

#define COLOR_MENU_BG_HL GColorBlack
#define COLOR_MENU_FG_HL GColorWhite
#define COLOR_MENU_BG GColorWhite
#define COLOR_MENU_FG GColorBlack

static Window *main_window, *scroll_window;
static MenuLayer *main_menu_layer;
static ScrollLayer *scroll_layer;
static TextLayer *scroll_title, *scroll_text;

struct menu_data_cell {
	char *title;
	char *subtitle;
	int16_t height;
};

static struct menu_data_cell data_cells[] = {
	{"Starbucks", "0.1 miles away", 3},
	{"BeechCafe", "0.3 miles away", 3},
	{"Duane Reade", "0.4 miles away", 3},
};

static void main_menu_draw_row(GContext *ctx, const Layer *cell_layer, MenuIndex *cell_index, void *menu_ctx) {
	struct menu_data_cell *data_cell;

	data_cell = &data_cells[cell_index->row];

	graphics_context_set_text_color(ctx, GColorBlack);
	graphics_draw_text(ctx, data_cell->title, fonts_get_system_font(FONT_KEY_GOTHIC_24_BOLD), GRect(7, -3, 130, 30),
			GTextOverflowModeTrailingEllipsis, GTextAlignmentLeft, NULL);
	graphics_draw_text(ctx, data_cell->subtitle, fonts_get_system_font(FONT_KEY_GOTHIC_14), GRect(7, 23, 120, 30),
			GTextOverflowModeTrailingEllipsis, GTextAlignmentLeft, NULL);
	graphics_draw_text(ctx, "\U0001F605", fonts_get_system_font(FONT_KEY_GOTHIC_24_BOLD), GRect(90, 5, 50, 50),
			GTextOverflowModeTrailingEllipsis, GTextAlignmentRight, NULL);
}

static int16_t main_menu_header_height(struct MenuLayer *menu_layer, uint16_t section_index, void *menu_ctx) {
	return 22;
}

static void main_menu_header(GContext *ctx, const Layer *cell_layer, uint16_t section_index, void *menu_ctx) {
	graphics_context_set_fill_color(ctx, GColorBlack);
	graphics_context_set_text_color(ctx, GColorWhite);
	graphics_fill_rect(ctx, layer_get_bounds(cell_layer), 0, GCornersAll);
	graphics_draw_text(ctx, "[Restrooms]", fonts_get_system_font(FONT_KEY_GOTHIC_18_BOLD),
			layer_get_bounds(cell_layer), GTextOverflowModeFill, GTextAlignmentCenter, NULL);
}

static uint16_t main_menu_num_rows(struct MenuLayer *menu_layer, uint16_t section_index, void *menu_ctx) {
	return ARRAY_LENGTH(data_cells);
}

static void main_menu_select_click(struct MenuLayer *menu_layer, MenuIndex *cell_index, void *menu_ctx) {
	window_stack_push(scroll_window, true);
}

static void main_window_load(Window *window) {
	Layer *window_layer;
	GRect window_bounds;

	window_set_background_color(window, COLOR_WINDOW_BG);

	window_layer = window_get_root_layer(window);
	window_bounds = layer_get_bounds(window_layer);

	APP_LOG(APP_LOG_LEVEL_INFO, "bounds of window: %d,%d:%d,%d",
			window_bounds.origin.x, window_bounds.origin.y,
			window_bounds.size.w, window_bounds.size.h);

	main_menu_layer = menu_layer_create(window_bounds);
#ifdef PBL_SDK_3
	menu_layer_set_highlight_colors(main_menu_layer, COLOR_MENU_BG, COLOR_MENU_FG);
	menu_layer_set_normal_colors(main_menu_layer, COLOR_MENU_BG_HL, COLOR_MENU_FG_HL);
#endif
	menu_layer_set_click_config_onto_window(main_menu_layer, window);
	layer_add_child(window_layer, menu_layer_get_layer(main_menu_layer));

	menu_layer_set_callbacks(main_menu_layer, NULL, (MenuLayerCallbacks) {
		.get_header_height = main_menu_header_height,
		.draw_header = main_menu_header,
		.draw_row = main_menu_draw_row,
		.get_num_rows = main_menu_num_rows,
		.select_click = main_menu_select_click,
	});
}

static void main_window_unload(Window *window) {
	menu_layer_destroy(main_menu_layer);
}

static void scroll_window_load(Window *window) {
	Layer *window_layer;
	GRect bounds;

	window_layer = window_get_root_layer(window);
	bounds = layer_get_bounds(window_layer);

	APP_LOG(APP_LOG_LEVEL_INFO, "bounds of window: %d,%d:%d,%d",
			bounds.origin.x, bounds.origin.y,
			bounds.size.w, bounds.size.h);

	scroll_layer = scroll_layer_create(bounds);
	scroll_layer_set_click_config_onto_window(scroll_layer, window);

	scroll_title = text_layer_create(GRect(2, 0, bounds.size.w, 30));
	text_layer_set_text(scroll_title, "Starbucks Cafe This");
	text_layer_set_text_color(scroll_title, GColorBlack);
	text_layer_set_text_alignment(scroll_title, GTextAlignmentLeft);
	text_layer_set_font(scroll_title, fonts_get_system_font(FONT_KEY_GOTHIC_24_BOLD));
	scroll_layer_add_child(scroll_layer, text_layer_get_layer(scroll_title));

	bounds = layer_get_bounds(text_layer_get_layer(scroll_title));
	scroll_text = text_layer_create(GRect(2, bounds.size.h, bounds.size.w, 200));
	text_layer_set_text(scroll_text, "There are two restrooms in this Starbucks cafe.");
	text_layer_set_font(scroll_text, fonts_get_system_font(FONT_KEY_GOTHIC_18_BOLD));

	bounds = layer_get_bounds(window_layer);
	scroll_layer_set_content_size(scroll_layer, GSize(bounds.size.w, bounds.size.h - 20));
	scroll_layer_add_child(scroll_layer, text_layer_get_layer(scroll_text));

	layer_add_child(window_layer, scroll_layer_get_layer(scroll_layer));
}

static void scroll_window_unload(Window *window) {
	text_layer_destroy(scroll_text);
	text_layer_destroy(scroll_title);
	scroll_layer_destroy(scroll_layer);
}

static void init() {
	main_window = window_create();

	window_set_window_handlers(main_window, (WindowHandlers) {
		.load = main_window_load,
		.unload = main_window_unload,
	});

	scroll_window = window_create();

	window_set_window_handlers(scroll_window, (WindowHandlers) {
		.load = scroll_window_load,
		.unload = scroll_window_unload,
	});

	window_stack_push(main_window, true);
}

static void deinit() {
	window_destroy(scroll_window);
	window_destroy(main_window);
}

int main(void) {
	init();
	app_event_loop();
	deinit();

	return 0;
}
