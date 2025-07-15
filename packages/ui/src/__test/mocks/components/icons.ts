import React from "react";

export const mockIconCommon = React.forwardRef<SVGSVGElement, any>(({ name, fill, decorative, size, ...props }, ref) => {
  // Handle aria-hidden logic
  let ariaHidden;
  if (props["aria-hidden"] !== undefined) {
    // Explicit aria-hidden takes precedence
    ariaHidden = props["aria-hidden"] ? "true" : null;
  } else {
    // Default based on decorative (handle both string and boolean)
    const isDecorative = decorative === true || decorative === "true";
    ariaHidden = !isDecorative && (decorative === false || decorative === "false") ? null : "true";
  }

  // Handle aria-label logic  
  let ariaLabel;
  const isDecorative = decorative === true || decorative === "true";
  if (isDecorative) {
    // If explicitly decorative, no label
    ariaLabel = null;
  } else if (!isDecorative && props["aria-label"]) {
    ariaLabel = props["aria-label"];
  } else {
    ariaLabel = null;
  }

  const filteredProps = { ...props };
  delete filteredProps["aria-hidden"];
  delete filteredProps["aria-label"];
  delete filteredProps["data-testid"];
  delete filteredProps["decorative"];

  return React.createElement("svg", {
    "data-testid": props["data-testid"] || "icon-common",
    "data-icon-type": "common",
    "data-icon-name": name,
    width: size !== null && size !== undefined ? size : 24,
    height: size !== null && size !== undefined ? size : 24,
    fill: fill || "currentColor",
    "aria-hidden": ariaHidden,
    "aria-label": ariaLabel,
    ref,
    ...filteredProps,
  }, React.createElement("use", { href: `common-sprite.svg#${name}` }));
});

// Create a generic icon component factory
const createIcon = (name: string) => React.forwardRef<SVGSVGElement, any>(({ size }, ref) => 
  React.createElement("span", { 
    "data-testid": `${name.toLowerCase()}-icon`,
    "data-size": size,
    ref,
  }, name));

// Create a generic text icon component factory  
const createTextIcon = (name: string) => React.forwardRef<SVGSVGElement, any>(({ size }, ref) => 
  React.createElement("span", { 
    "data-testid": `${name.toLowerCase()}-text-icon`,
    "data-size": size,
    ref,
  }, `${name} Text`));

export const iconsMock = {
  // Common icons
  IconCommon: mockIconCommon,
  
  // Routine icons
  IconRoutine: React.forwardRef<SVGSVGElement, any>(({ name, size, fill, decorative, ...props }, ref) => {
    // Filter out non-DOM props
    const filteredProps = { ...props };
    delete filteredProps["data-testid"];
    
    return React.createElement("svg", { 
      "data-testid": props["data-testid"] || "icon-routine",
      "data-icon-type": "routine",
      "data-icon-name": name,
      width: size !== null && size !== undefined ? size : 24,
      height: size !== null && size !== undefined ? size : 24,
      fill: fill || "currentColor",
      ref,
      ...filteredProps,
    }, React.createElement("use", { href: `routine-sprite.svg#${name}` }));
  }),
  
  // Service icons
  IconService: React.forwardRef<SVGSVGElement, any>(({ name, size, fill, decorative, ...props }, ref) => {
    // Filter out non-DOM props
    const filteredProps = { ...props };
    delete filteredProps["data-testid"];
    
    return React.createElement("svg", { 
      "data-testid": props["data-testid"] || "icon-service",
      "data-icon-type": "service",
      "data-icon-name": name,
      width: size !== null && size !== undefined ? size : 24,
      height: size !== null && size !== undefined ? size : 24,
      fill: fill || "currentColor",
      ref,
      ...filteredProps,
    }, React.createElement("use", { href: `service-sprite.svg#${name}` }));
  }),
  
  // Text icons (Icon + Text label)
  IconText: React.forwardRef<SVGSVGElement, any>(({ name, size, fill, decorative, ...props }, ref) => {
    // Filter out non-DOM props
    const filteredProps = { ...props };
    delete filteredProps["data-testid"];
    
    return React.createElement("svg", { 
      "data-testid": props["data-testid"] || "icon-text",
      "data-icon-type": "text",
      "data-icon-name": name,
      width: size !== null && size !== undefined ? size : 24,
      height: size !== null && size !== undefined ? size : 24,
      fill: fill || "currentColor",
      ref,
      ...filteredProps,
    }, React.createElement("use", { href: `text-sprite.svg#${name}` }));
  }),
  
  // Individual icon components - all are created using the forwardRef-enabled factory
  AddIcon: createIcon("Add"),
  CloseIcon: createIcon("Close"),
  EditIcon: createIcon("Edit"),
  DeleteIcon: createIcon("Delete"),
  SaveIcon: createIcon("Save"),
  CancelIcon: createIcon("Cancel"),
  SearchIcon: createIcon("Search"),
  MenuIcon: createIcon("Menu"),
  SettingsIcon: createIcon("Settings"),
  UserIcon: createIcon("User"),
  HomeIcon: createIcon("Home"),
  BackIcon: createIcon("Back"),
  ForwardIcon: createIcon("Forward"),
  UpIcon: createIcon("Up"),
  DownIcon: createIcon("Down"),
  ListIcon: createIcon("List"),
  GridIcon: createIcon("Grid"),
  FilterIcon: createIcon("Filter"),
  SortIcon: createIcon("Sort"),
  InfoIcon: createIcon("Info"),
  WarningIcon: createIcon("Warning"),
  ErrorIcon: createIcon("Error"),
  SuccessIcon: createIcon("Success"),
  LockIcon: createIcon("Lock"),
  UnlockIcon: createIcon("Unlock"),
  VisibleIcon: createIcon("Visible"),
  InvisibleIcon: createIcon("Invisible"),
  CopyIcon: createIcon("Copy"),
  PasteIcon: createIcon("Paste"),
  RefreshIcon: createIcon("Refresh"),
  ShareIcon: createIcon("Share"),
  DownloadIcon: createIcon("Download"),
  UploadIcon: createIcon("Upload"),
  HeartIcon: createIcon("Heart"),
  StarIcon: createIcon("Star"),
  CommentIcon: createIcon("Comment"),
  NotificationIcon: createIcon("Notification"),
  CalendarIcon: createIcon("Calendar"),
  ClockIcon: createIcon("Clock"),
  LocationIcon: createIcon("Location"),
  PhoneIcon: createIcon("Phone"),
  EmailIcon: createIcon("Email"),
  LinkIcon: createIcon("Link"),
  AttachmentIcon: createIcon("Attachment"),
  FolderIcon: createIcon("Folder"),
  FileIcon: createIcon("File"),
  ImageIcon: createIcon("Image"),
  VideoIcon: createIcon("Video"),
  AudioIcon: createIcon("Audio"),
  DocumentIcon: createIcon("Document"),
  CodeIcon: createIcon("Code"),
  TerminalIcon: createIcon("Terminal"),
  BugIcon: createIcon("Bug"),
  RocketIcon: createIcon("Rocket"),
  TrophyIcon: createIcon("Trophy"),
  FlagIcon: createIcon("Flag"),
  BookmarkIcon: createIcon("Bookmark"),
  TagIcon: createIcon("Tag"),
  PrintIcon: createIcon("Print"),
  CameraIcon: createIcon("Camera"),
  MicrophoneIcon: createIcon("Microphone"),
  PlayIcon: createIcon("Play"),
  PauseIcon: createIcon("Pause"),
  StopIcon: createIcon("Stop"),
  SkipIcon: createIcon("Skip"),
  RewindIcon: createIcon("Rewind"),
  VolumeIcon: createIcon("Volume"),
  MuteIcon: createIcon("Mute"),
  FullscreenIcon: createIcon("Fullscreen"),
  MinimizeIcon: createIcon("Minimize"),
  MaximizeIcon: createIcon("Maximize"),
  RestoreIcon: createIcon("Restore"),
  
  // Special favicon icon component
  IconFavicon: React.forwardRef<HTMLElement, any>(({ href, size, fallbackIcon, ...props }, ref) => {
    // Simple URL validation for testing
    const isValidUrl = href && (href.startsWith("http://") || href.startsWith("https://"));
    
    if (isValidUrl) {
      // Valid URL - render as img
      let domain;
      try {
        domain = new URL(href).hostname;
      } catch (e) {
        console.error(`[IconFavicon] Invalid URL: ${href}`, e);
        domain = "unknown";
      }
      
      return React.createElement("img", { 
        "data-testid": props["data-testid"] || "favicon-icon",
        "data-icon-type": "favicon",
        "data-favicon-domain": domain,
        "src": href,
        "alt": props["aria-label"] || "favicon",
        "width": size || 16,
        "height": size || 16,
        className: props.className,
        ref: ref as React.Ref<HTMLImageElement>,
      });
    } else {
      // Invalid URL - render fallback SVG
      console.error(`[IconFavicon] Invalid URL: ${href}`, new Error());
      
      const fallbackName = fallbackIcon?.name || "Website";
      const fallbackType = fallbackIcon ? "favicon-fallback" : "favicon";
      
      // Filter out decorative prop before passing to SVG
      const filteredFallbackProps = { ...props };
      delete filteredFallbackProps["decorative"];
      delete filteredFallbackProps["data-testid"];
      
      return React.createElement("svg", {
        "data-testid": props["data-testid"] || "favicon-icon",
        "data-icon-type": fallbackType,
        "data-icon-name": fallbackName,
        width: size || 16,
        height: size || 16,
        fill: "currentColor",
        "aria-hidden": !(props.decorative === false || props.decorative === "false") ? "true" : null,
        ref: ref as React.Ref<SVGSVGElement>,
        ...filteredFallbackProps,
      }, React.createElement("use", { href: `#${fallbackName}` }));
    }
  }),
  
  // Unified Icon component that takes IconInfo
  Icon: React.forwardRef<SVGSVGElement, any>(({ info, size, fill, sizeOverride, ...props }, ref) => {
    if (!info || !info.name || !info.type) {
      console.error("Invalid icon info", info, "Stack:", new Error().stack);
      return null;
    }
    
    const typeMap = {
      "Common": "common",
      "Routine": "routine", 
      "Service": "service",
      "Text": "text",
    };
    
    const iconType = typeMap[info.type] || "common";
    
    // Filter out props that shouldn't be passed to SVG element
    const filteredIconProps = { ...props };
    delete filteredIconProps["decorative"];
    delete filteredIconProps["data-testid"];
    delete filteredIconProps["aria-label"];
    delete filteredIconProps["info"];
    
    // Use sizeOverride if provided, otherwise use size
    const finalSize = sizeOverride !== null && sizeOverride !== undefined ? sizeOverride : size;
    
    return React.createElement("svg", {
      "data-testid": props["data-testid"] || "icon",
      "data-icon-type": iconType,
      "data-icon-name": info.name,
      width: finalSize !== null && finalSize !== undefined ? finalSize : 24,
      height: finalSize !== null && finalSize !== undefined ? finalSize : 24,
      fill: fill || "currentColor",
      "aria-hidden": !(props.decorative === false || props.decorative === "false") ? "true" : null,
      "aria-label": !(props.decorative === true || props.decorative === "true") && props["aria-label"] ? props["aria-label"] : null,
      ref,
      ...filteredIconProps,
    }, React.createElement("use", { href: `${iconType}-sprite.svg#${info.name}` }));
  }),

  // Catch-all for any icons we might have missed
  GenericIcon: React.forwardRef<SVGSVGElement, any>(({ name }, ref) => 
    React.createElement(createIcon(name || "Icon"), { ref })),
};

// Set display names for the components to match the test expectations
iconsMock.IconCommon.displayName = "IconCommon";
iconsMock.IconRoutine.displayName = "IconRoutine";
iconsMock.IconService.displayName = "IconService";
iconsMock.IconText.displayName = "IconText";
iconsMock.Icon.displayName = "Icon";
iconsMock.IconFavicon.displayName = "IconFavicon";
