/**
 * Displays version of object.
 * On hover or press, a popup displays a list of all versions. 
 * On click, the version is loaded.
 */
 import { Box, LinearProgress, Popover, Typography, useTheme } from "@mui/material";
 import { VersionDisplayProps } from "../types";
 import { Today as CalendarIcon } from "@mui/icons-material";
 import { displayDate } from "utils";
 import { useCallback, useState } from "react";
 
 export const VersionDisplay = ({
    handleVersionChange,
     loading = false,
     showIcon = true,
     versions = [],
     ...props
 }: VersionDisplayProps) => {
    return null;
    //  const { palette } = useTheme();
    //  const shadowColor = palette.mode === 'light' ? '0 0 0' : '255 255 255';
 
    //  // Full date popup
    //  const [anchorEl, setAnchorEl] = useState<any | null>(null);
    //  const isOpen = Boolean(anchorEl);
    //  const open = useCallback((ev: React.MouseEvent | React.TouchEvent) => {
    //      ev.preventDefault();
    //      setAnchorEl(ev.currentTarget ?? ev.target)
    //  }, []);
    //  const close = useCallback(() => setAnchorEl(null), []);
 
    //  if (loading) return (
    //      <Box {...props}>
    //          <LinearProgress color="inherit" sx={{ height: '6px', borderRadius: '12px' }} />
    //      </Box>
    //  );
    //  if (!timestamp) return null;
    //  return (
    //      <>
    //          {/* Full date popup */}
    //          <Popover
    //              open={isOpen}
    //              anchorEl={anchorEl}
    //              onClose={close}
    //              anchorOrigin={{
    //                  vertical: 'top',
    //                  horizontal: 'center',
    //              }}
    //              transformOrigin={{
    //                  vertical: 'bottom',
    //                  horizontal: 'center',
    //              }}
    //              sx={{
    //                  '& .MuiPopover-paper': {
    //                      padding: 1,
    //                      overflow: 'unset',
    //                      background: palette.background.paper,
    //                      color: palette.background.textPrimary,
    //                      boxShadow: `0px 5px 5px -3px rgb(${shadowColor} / 20%), 
    //                      0px 8px 10px 1px rgb(${shadowColor} / 14%), 
    //                      0px 3px 14px 2px rgb(${shadowColor} / 12%)`
    //                  }
    //              }}
    //          >
    //              <Box>
    //                  <Typography variant="body2" color="textSecondary">
    //                      {displayDate(timestamp, true)}
    //                  </Typography>
    //                  {/* Triangle placed below popper */}
    //                  <Box sx={{
    //                      width: '0',
    //                      height: '0',
    //                      borderLeft: '10px solid transparent',
    //                      borderRight: '10px solid transparent',
    //                      borderTop: `10px solid ${palette.background.paper}`,
    //                      position: 'absolute',
    //                      bottom: '-10px',
    //                      left: '50%',
    //                      transform: 'translateX(-50%)',
    //                  }} />
 
    //              </Box>
    //          </Popover>
    //          {/* Displayed date */}
    //          <Box
    //              {...props}
    //              display="flex"
    //              justifyContent="center"
    //              onClick={open}
    //              sx={{
    //                  ...(props.sx ?? {}),
    //                  cursor: 'pointer',
    //              }}
    //          >
    //              {showIcon && <CalendarIcon />}
    //              {`${textBeforeDate} ${displayDate(timestamp, false)}`}
    //          </Box>
    //      </>
    //  )
 }