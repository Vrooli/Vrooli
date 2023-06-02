export const auth_emailLogIn = {
  "fieldName": "emailLogIn",
  "fieldNodes": [
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "emailLogIn",
        "loc": {
          "start": 326,
          "end": 336
        }
      },
      "arguments": [
        {
          "kind": "Argument",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 337,
              "end": 342
            }
          },
          "value": {
            "kind": "Variable",
            "name": {
              "kind": "Name",
              "value": "input",
              "loc": {
                "start": 345,
                "end": 350
              }
            },
            "loc": {
              "start": 344,
              "end": 350
            }
          },
          "loc": {
            "start": 337,
            "end": 350
          }
        }
      ],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLoggedIn",
              "loc": {
                "start": 358,
                "end": 368
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 358,
              "end": 368
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timeZone",
              "loc": {
                "start": 373,
                "end": 381
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 373,
              "end": 381
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "users",
              "loc": {
                "start": 386,
                "end": 391
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "activeFocusMode",
                    "loc": {
                      "start": 402,
                      "end": 417
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "mode",
                          "loc": {
                            "start": 432,
                            "end": 436
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "filters",
                                "loc": {
                                  "start": 455,
                                  "end": 462
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 485,
                                        "end": 487
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 485,
                                      "end": 487
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "filterType",
                                      "loc": {
                                        "start": 508,
                                        "end": 518
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 508,
                                      "end": 518
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "tag",
                                      "loc": {
                                        "start": 539,
                                        "end": 542
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 569,
                                              "end": 571
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 569,
                                            "end": 571
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 596,
                                              "end": 606
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 596,
                                            "end": 606
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "tag",
                                            "loc": {
                                              "start": 631,
                                              "end": 634
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 631,
                                            "end": 634
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "bookmarks",
                                            "loc": {
                                              "start": 659,
                                              "end": 668
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 659,
                                            "end": 668
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "translations",
                                            "loc": {
                                              "start": 693,
                                              "end": 705
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "id",
                                                  "loc": {
                                                    "start": 736,
                                                    "end": 738
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 736,
                                                  "end": 738
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "language",
                                                  "loc": {
                                                    "start": 767,
                                                    "end": 775
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 767,
                                                  "end": 775
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "description",
                                                  "loc": {
                                                    "start": 804,
                                                    "end": 815
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 804,
                                                  "end": 815
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 706,
                                              "end": 841
                                            }
                                          },
                                          "loc": {
                                            "start": 693,
                                            "end": 841
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "you",
                                            "loc": {
                                              "start": 866,
                                              "end": 869
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isOwn",
                                                  "loc": {
                                                    "start": 900,
                                                    "end": 905
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 900,
                                                  "end": 905
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isBookmarked",
                                                  "loc": {
                                                    "start": 934,
                                                    "end": 946
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 934,
                                                  "end": 946
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 870,
                                              "end": 972
                                            }
                                          },
                                          "loc": {
                                            "start": 866,
                                            "end": 972
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 543,
                                        "end": 994
                                      }
                                    },
                                    "loc": {
                                      "start": 539,
                                      "end": 994
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "focusMode",
                                      "loc": {
                                        "start": 1015,
                                        "end": 1024
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "labels",
                                            "loc": {
                                              "start": 1051,
                                              "end": 1057
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "id",
                                                  "loc": {
                                                    "start": 1088,
                                                    "end": 1090
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1088,
                                                  "end": 1090
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "color",
                                                  "loc": {
                                                    "start": 1119,
                                                    "end": 1124
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1119,
                                                  "end": 1124
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "label",
                                                  "loc": {
                                                    "start": 1153,
                                                    "end": 1158
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1153,
                                                  "end": 1158
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 1058,
                                              "end": 1184
                                            }
                                          },
                                          "loc": {
                                            "start": 1051,
                                            "end": 1184
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "schedule",
                                            "loc": {
                                              "start": 1209,
                                              "end": 1217
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "FragmentSpread",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "Schedule_common",
                                                  "loc": {
                                                    "start": 1251,
                                                    "end": 1266
                                                  }
                                                },
                                                "directives": [],
                                                "loc": {
                                                  "start": 1248,
                                                  "end": 1266
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 1218,
                                              "end": 1292
                                            }
                                          },
                                          "loc": {
                                            "start": 1209,
                                            "end": 1292
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 1317,
                                              "end": 1319
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1317,
                                            "end": 1319
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 1344,
                                              "end": 1348
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1344,
                                            "end": 1348
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 1373,
                                              "end": 1384
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1373,
                                            "end": 1384
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 1025,
                                        "end": 1406
                                      }
                                    },
                                    "loc": {
                                      "start": 1015,
                                      "end": 1406
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 463,
                                  "end": 1424
                                }
                              },
                              "loc": {
                                "start": 455,
                                "end": 1424
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "labels",
                                "loc": {
                                  "start": 1441,
                                  "end": 1447
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 1470,
                                        "end": 1472
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1470,
                                      "end": 1472
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "color",
                                      "loc": {
                                        "start": 1493,
                                        "end": 1498
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1493,
                                      "end": 1498
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "label",
                                      "loc": {
                                        "start": 1519,
                                        "end": 1524
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1519,
                                      "end": 1524
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 1448,
                                  "end": 1542
                                }
                              },
                              "loc": {
                                "start": 1441,
                                "end": 1542
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "reminderList",
                                "loc": {
                                  "start": 1559,
                                  "end": 1571
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 1594,
                                        "end": 1596
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1594,
                                      "end": 1596
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 1617,
                                        "end": 1627
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1617,
                                      "end": 1627
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 1648,
                                        "end": 1658
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1648,
                                      "end": 1658
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "reminders",
                                      "loc": {
                                        "start": 1679,
                                        "end": 1688
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 1715,
                                              "end": 1717
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1715,
                                            "end": 1717
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 1742,
                                              "end": 1752
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1742,
                                            "end": 1752
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "updated_at",
                                            "loc": {
                                              "start": 1777,
                                              "end": 1787
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1777,
                                            "end": 1787
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 1812,
                                              "end": 1816
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1812,
                                            "end": 1816
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 1841,
                                              "end": 1852
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1841,
                                            "end": 1852
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "dueDate",
                                            "loc": {
                                              "start": 1877,
                                              "end": 1884
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1877,
                                            "end": 1884
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "index",
                                            "loc": {
                                              "start": 1909,
                                              "end": 1914
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1909,
                                            "end": 1914
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isComplete",
                                            "loc": {
                                              "start": 1939,
                                              "end": 1949
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1939,
                                            "end": 1949
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "reminderItems",
                                            "loc": {
                                              "start": 1974,
                                              "end": 1987
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "id",
                                                  "loc": {
                                                    "start": 2018,
                                                    "end": 2020
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2018,
                                                  "end": 2020
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "created_at",
                                                  "loc": {
                                                    "start": 2049,
                                                    "end": 2059
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2049,
                                                  "end": 2059
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "updated_at",
                                                  "loc": {
                                                    "start": 2088,
                                                    "end": 2098
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2088,
                                                  "end": 2098
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "name",
                                                  "loc": {
                                                    "start": 2127,
                                                    "end": 2131
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2127,
                                                  "end": 2131
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "description",
                                                  "loc": {
                                                    "start": 2160,
                                                    "end": 2171
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2160,
                                                  "end": 2171
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "dueDate",
                                                  "loc": {
                                                    "start": 2200,
                                                    "end": 2207
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2200,
                                                  "end": 2207
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "index",
                                                  "loc": {
                                                    "start": 2236,
                                                    "end": 2241
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2236,
                                                  "end": 2241
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isComplete",
                                                  "loc": {
                                                    "start": 2270,
                                                    "end": 2280
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2270,
                                                  "end": 2280
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 1988,
                                              "end": 2306
                                            }
                                          },
                                          "loc": {
                                            "start": 1974,
                                            "end": 2306
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 1689,
                                        "end": 2328
                                      }
                                    },
                                    "loc": {
                                      "start": 1679,
                                      "end": 2328
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 1572,
                                  "end": 2346
                                }
                              },
                              "loc": {
                                "start": 1559,
                                "end": 2346
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "resourceList",
                                "loc": {
                                  "start": 2363,
                                  "end": 2375
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 2398,
                                        "end": 2400
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2398,
                                      "end": 2400
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 2421,
                                        "end": 2431
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2421,
                                      "end": 2431
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "translations",
                                      "loc": {
                                        "start": 2452,
                                        "end": 2464
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 2491,
                                              "end": 2493
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2491,
                                            "end": 2493
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "language",
                                            "loc": {
                                              "start": 2518,
                                              "end": 2526
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2518,
                                            "end": 2526
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 2551,
                                              "end": 2562
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2551,
                                            "end": 2562
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 2587,
                                              "end": 2591
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2587,
                                            "end": 2591
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 2465,
                                        "end": 2613
                                      }
                                    },
                                    "loc": {
                                      "start": 2452,
                                      "end": 2613
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "resources",
                                      "loc": {
                                        "start": 2634,
                                        "end": 2643
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 2670,
                                              "end": 2672
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2670,
                                            "end": 2672
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "index",
                                            "loc": {
                                              "start": 2697,
                                              "end": 2702
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2697,
                                            "end": 2702
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "link",
                                            "loc": {
                                              "start": 2727,
                                              "end": 2731
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2727,
                                            "end": 2731
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "usedFor",
                                            "loc": {
                                              "start": 2756,
                                              "end": 2763
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2756,
                                            "end": 2763
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "translations",
                                            "loc": {
                                              "start": 2788,
                                              "end": 2800
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "id",
                                                  "loc": {
                                                    "start": 2831,
                                                    "end": 2833
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2831,
                                                  "end": 2833
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "language",
                                                  "loc": {
                                                    "start": 2862,
                                                    "end": 2870
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2862,
                                                  "end": 2870
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "description",
                                                  "loc": {
                                                    "start": 2899,
                                                    "end": 2910
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2899,
                                                  "end": 2910
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "name",
                                                  "loc": {
                                                    "start": 2939,
                                                    "end": 2943
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2939,
                                                  "end": 2943
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 2801,
                                              "end": 2969
                                            }
                                          },
                                          "loc": {
                                            "start": 2788,
                                            "end": 2969
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 2644,
                                        "end": 2991
                                      }
                                    },
                                    "loc": {
                                      "start": 2634,
                                      "end": 2991
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 2376,
                                  "end": 3009
                                }
                              },
                              "loc": {
                                "start": 2363,
                                "end": 3009
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "schedule",
                                "loc": {
                                  "start": 3026,
                                  "end": 3034
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "FragmentSpread",
                                    "name": {
                                      "kind": "Name",
                                      "value": "Schedule_common",
                                      "loc": {
                                        "start": 3060,
                                        "end": 3075
                                      }
                                    },
                                    "directives": [],
                                    "loc": {
                                      "start": 3057,
                                      "end": 3075
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 3035,
                                  "end": 3093
                                }
                              },
                              "loc": {
                                "start": 3026,
                                "end": 3093
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 3110,
                                  "end": 3112
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3110,
                                "end": 3112
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 3129,
                                  "end": 3133
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3129,
                                "end": 3133
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 3150,
                                  "end": 3161
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3150,
                                "end": 3161
                              }
                            }
                          ],
                          "loc": {
                            "start": 437,
                            "end": 3175
                          }
                        },
                        "loc": {
                          "start": 432,
                          "end": 3175
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "stopCondition",
                          "loc": {
                            "start": 3188,
                            "end": 3201
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3188,
                          "end": 3201
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "stopTime",
                          "loc": {
                            "start": 3214,
                            "end": 3222
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3214,
                          "end": 3222
                        }
                      }
                    ],
                    "loc": {
                      "start": 418,
                      "end": 3232
                    }
                  },
                  "loc": {
                    "start": 402,
                    "end": 3232
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "apisCount",
                    "loc": {
                      "start": 3241,
                      "end": 3250
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3241,
                    "end": 3250
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "bookmarkLists",
                    "loc": {
                      "start": 3259,
                      "end": 3272
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 3287,
                            "end": 3289
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3287,
                          "end": 3289
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 3302,
                            "end": 3312
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3302,
                          "end": 3312
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 3325,
                            "end": 3335
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3325,
                          "end": 3335
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "label",
                          "loc": {
                            "start": 3348,
                            "end": 3353
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3348,
                          "end": 3353
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "bookmarksCount",
                          "loc": {
                            "start": 3366,
                            "end": 3380
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3366,
                          "end": 3380
                        }
                      }
                    ],
                    "loc": {
                      "start": 3273,
                      "end": 3390
                    }
                  },
                  "loc": {
                    "start": 3259,
                    "end": 3390
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "focusModes",
                    "loc": {
                      "start": 3399,
                      "end": 3409
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "filters",
                          "loc": {
                            "start": 3424,
                            "end": 3431
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 3450,
                                  "end": 3452
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3450,
                                "end": 3452
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "filterType",
                                "loc": {
                                  "start": 3469,
                                  "end": 3479
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3469,
                                "end": 3479
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "tag",
                                "loc": {
                                  "start": 3496,
                                  "end": 3499
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 3522,
                                        "end": 3524
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3522,
                                      "end": 3524
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 3545,
                                        "end": 3555
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3545,
                                      "end": 3555
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "tag",
                                      "loc": {
                                        "start": 3576,
                                        "end": 3579
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3576,
                                      "end": 3579
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "bookmarks",
                                      "loc": {
                                        "start": 3600,
                                        "end": 3609
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3600,
                                      "end": 3609
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "translations",
                                      "loc": {
                                        "start": 3630,
                                        "end": 3642
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 3669,
                                              "end": 3671
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3669,
                                            "end": 3671
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "language",
                                            "loc": {
                                              "start": 3696,
                                              "end": 3704
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3696,
                                            "end": 3704
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 3729,
                                              "end": 3740
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3729,
                                            "end": 3740
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 3643,
                                        "end": 3762
                                      }
                                    },
                                    "loc": {
                                      "start": 3630,
                                      "end": 3762
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "you",
                                      "loc": {
                                        "start": 3783,
                                        "end": 3786
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isOwn",
                                            "loc": {
                                              "start": 3813,
                                              "end": 3818
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3813,
                                            "end": 3818
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isBookmarked",
                                            "loc": {
                                              "start": 3843,
                                              "end": 3855
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3843,
                                            "end": 3855
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 3787,
                                        "end": 3877
                                      }
                                    },
                                    "loc": {
                                      "start": 3783,
                                      "end": 3877
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 3500,
                                  "end": 3895
                                }
                              },
                              "loc": {
                                "start": 3496,
                                "end": 3895
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "focusMode",
                                "loc": {
                                  "start": 3912,
                                  "end": 3921
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "labels",
                                      "loc": {
                                        "start": 3944,
                                        "end": 3950
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 3977,
                                              "end": 3979
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3977,
                                            "end": 3979
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "color",
                                            "loc": {
                                              "start": 4004,
                                              "end": 4009
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4004,
                                            "end": 4009
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "label",
                                            "loc": {
                                              "start": 4034,
                                              "end": 4039
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4034,
                                            "end": 4039
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 3951,
                                        "end": 4061
                                      }
                                    },
                                    "loc": {
                                      "start": 3944,
                                      "end": 4061
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "schedule",
                                      "loc": {
                                        "start": 4082,
                                        "end": 4090
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "FragmentSpread",
                                          "name": {
                                            "kind": "Name",
                                            "value": "Schedule_common",
                                            "loc": {
                                              "start": 4120,
                                              "end": 4135
                                            }
                                          },
                                          "directives": [],
                                          "loc": {
                                            "start": 4117,
                                            "end": 4135
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 4091,
                                        "end": 4157
                                      }
                                    },
                                    "loc": {
                                      "start": 4082,
                                      "end": 4157
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 4178,
                                        "end": 4180
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4178,
                                      "end": 4180
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 4201,
                                        "end": 4205
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4201,
                                      "end": 4205
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 4226,
                                        "end": 4237
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4226,
                                      "end": 4237
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 3922,
                                  "end": 4255
                                }
                              },
                              "loc": {
                                "start": 3912,
                                "end": 4255
                              }
                            }
                          ],
                          "loc": {
                            "start": 3432,
                            "end": 4269
                          }
                        },
                        "loc": {
                          "start": 3424,
                          "end": 4269
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "labels",
                          "loc": {
                            "start": 4282,
                            "end": 4288
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 4307,
                                  "end": 4309
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4307,
                                "end": 4309
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "color",
                                "loc": {
                                  "start": 4326,
                                  "end": 4331
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4326,
                                "end": 4331
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "label",
                                "loc": {
                                  "start": 4348,
                                  "end": 4353
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4348,
                                "end": 4353
                              }
                            }
                          ],
                          "loc": {
                            "start": 4289,
                            "end": 4367
                          }
                        },
                        "loc": {
                          "start": 4282,
                          "end": 4367
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "reminderList",
                          "loc": {
                            "start": 4380,
                            "end": 4392
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 4411,
                                  "end": 4413
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4411,
                                "end": 4413
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 4430,
                                  "end": 4440
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4430,
                                "end": 4440
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "updated_at",
                                "loc": {
                                  "start": 4457,
                                  "end": 4467
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4457,
                                "end": 4467
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "reminders",
                                "loc": {
                                  "start": 4484,
                                  "end": 4493
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 4516,
                                        "end": 4518
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4516,
                                      "end": 4518
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 4539,
                                        "end": 4549
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4539,
                                      "end": 4549
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 4570,
                                        "end": 4580
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4570,
                                      "end": 4580
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 4601,
                                        "end": 4605
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4601,
                                      "end": 4605
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 4626,
                                        "end": 4637
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4626,
                                      "end": 4637
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "dueDate",
                                      "loc": {
                                        "start": 4658,
                                        "end": 4665
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4658,
                                      "end": 4665
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "index",
                                      "loc": {
                                        "start": 4686,
                                        "end": 4691
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4686,
                                      "end": 4691
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isComplete",
                                      "loc": {
                                        "start": 4712,
                                        "end": 4722
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4712,
                                      "end": 4722
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "reminderItems",
                                      "loc": {
                                        "start": 4743,
                                        "end": 4756
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 4783,
                                              "end": 4785
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4783,
                                            "end": 4785
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 4810,
                                              "end": 4820
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4810,
                                            "end": 4820
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "updated_at",
                                            "loc": {
                                              "start": 4845,
                                              "end": 4855
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4845,
                                            "end": 4855
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 4880,
                                              "end": 4884
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4880,
                                            "end": 4884
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 4909,
                                              "end": 4920
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4909,
                                            "end": 4920
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "dueDate",
                                            "loc": {
                                              "start": 4945,
                                              "end": 4952
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4945,
                                            "end": 4952
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "index",
                                            "loc": {
                                              "start": 4977,
                                              "end": 4982
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4977,
                                            "end": 4982
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isComplete",
                                            "loc": {
                                              "start": 5007,
                                              "end": 5017
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5007,
                                            "end": 5017
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 4757,
                                        "end": 5039
                                      }
                                    },
                                    "loc": {
                                      "start": 4743,
                                      "end": 5039
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 4494,
                                  "end": 5057
                                }
                              },
                              "loc": {
                                "start": 4484,
                                "end": 5057
                              }
                            }
                          ],
                          "loc": {
                            "start": 4393,
                            "end": 5071
                          }
                        },
                        "loc": {
                          "start": 4380,
                          "end": 5071
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "resourceList",
                          "loc": {
                            "start": 5084,
                            "end": 5096
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 5115,
                                  "end": 5117
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5115,
                                "end": 5117
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 5134,
                                  "end": 5144
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5134,
                                "end": 5144
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "translations",
                                "loc": {
                                  "start": 5161,
                                  "end": 5173
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 5196,
                                        "end": 5198
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5196,
                                      "end": 5198
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "language",
                                      "loc": {
                                        "start": 5219,
                                        "end": 5227
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5219,
                                      "end": 5227
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 5248,
                                        "end": 5259
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5248,
                                      "end": 5259
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 5280,
                                        "end": 5284
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5280,
                                      "end": 5284
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 5174,
                                  "end": 5302
                                }
                              },
                              "loc": {
                                "start": 5161,
                                "end": 5302
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "resources",
                                "loc": {
                                  "start": 5319,
                                  "end": 5328
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 5351,
                                        "end": 5353
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5351,
                                      "end": 5353
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "index",
                                      "loc": {
                                        "start": 5374,
                                        "end": 5379
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5374,
                                      "end": 5379
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "link",
                                      "loc": {
                                        "start": 5400,
                                        "end": 5404
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5400,
                                      "end": 5404
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "usedFor",
                                      "loc": {
                                        "start": 5425,
                                        "end": 5432
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5425,
                                      "end": 5432
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "translations",
                                      "loc": {
                                        "start": 5453,
                                        "end": 5465
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 5492,
                                              "end": 5494
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5492,
                                            "end": 5494
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "language",
                                            "loc": {
                                              "start": 5519,
                                              "end": 5527
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5519,
                                            "end": 5527
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 5552,
                                              "end": 5563
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5552,
                                            "end": 5563
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 5588,
                                              "end": 5592
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5588,
                                            "end": 5592
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 5466,
                                        "end": 5614
                                      }
                                    },
                                    "loc": {
                                      "start": 5453,
                                      "end": 5614
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 5329,
                                  "end": 5632
                                }
                              },
                              "loc": {
                                "start": 5319,
                                "end": 5632
                              }
                            }
                          ],
                          "loc": {
                            "start": 5097,
                            "end": 5646
                          }
                        },
                        "loc": {
                          "start": 5084,
                          "end": 5646
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "schedule",
                          "loc": {
                            "start": 5659,
                            "end": 5667
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "FragmentSpread",
                              "name": {
                                "kind": "Name",
                                "value": "Schedule_common",
                                "loc": {
                                  "start": 5689,
                                  "end": 5704
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 5686,
                                "end": 5704
                              }
                            }
                          ],
                          "loc": {
                            "start": 5668,
                            "end": 5718
                          }
                        },
                        "loc": {
                          "start": 5659,
                          "end": 5718
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 5731,
                            "end": 5733
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5731,
                          "end": 5733
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 5746,
                            "end": 5750
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5746,
                          "end": 5750
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 5763,
                            "end": 5774
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5763,
                          "end": 5774
                        }
                      }
                    ],
                    "loc": {
                      "start": 3410,
                      "end": 5784
                    }
                  },
                  "loc": {
                    "start": 3399,
                    "end": 5784
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "handle",
                    "loc": {
                      "start": 5793,
                      "end": 5799
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5793,
                    "end": 5799
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "hasPremium",
                    "loc": {
                      "start": 5808,
                      "end": 5818
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5808,
                    "end": 5818
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 5827,
                      "end": 5829
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5827,
                    "end": 5829
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "languages",
                    "loc": {
                      "start": 5838,
                      "end": 5847
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5838,
                    "end": 5847
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "membershipsCount",
                    "loc": {
                      "start": 5856,
                      "end": 5872
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5856,
                    "end": 5872
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 5881,
                      "end": 5885
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5881,
                    "end": 5885
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "notesCount",
                    "loc": {
                      "start": 5894,
                      "end": 5904
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5894,
                    "end": 5904
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "projectsCount",
                    "loc": {
                      "start": 5913,
                      "end": 5926
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5913,
                    "end": 5926
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "questionsAskedCount",
                    "loc": {
                      "start": 5935,
                      "end": 5954
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5935,
                    "end": 5954
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "routinesCount",
                    "loc": {
                      "start": 5963,
                      "end": 5976
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5963,
                    "end": 5976
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "smartContractsCount",
                    "loc": {
                      "start": 5985,
                      "end": 6004
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5985,
                    "end": 6004
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "standardsCount",
                    "loc": {
                      "start": 6013,
                      "end": 6027
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6013,
                    "end": 6027
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "theme",
                    "loc": {
                      "start": 6036,
                      "end": 6041
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6036,
                    "end": 6041
                  }
                }
              ],
              "loc": {
                "start": 392,
                "end": 6047
              }
            },
            "loc": {
              "start": 386,
              "end": 6047
            }
          }
        ],
        "loc": {
          "start": 352,
          "end": 6051
        }
      },
      "loc": {
        "start": 326,
        "end": 6051
      }
    }
  ],
  "returnType": null,
  "parentType": null,
  "schema": null,
  "fragments": {
    "Schedule_common": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Schedule_common",
        "loc": {
          "start": 9,
          "end": 24
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Schedule",
          "loc": {
            "start": 28,
            "end": 36
          }
        },
        "loc": {
          "start": 28,
          "end": 36
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 39,
                "end": 41
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 39,
              "end": 41
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 42,
                "end": 52
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 42,
              "end": 52
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 53,
                "end": 63
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 53,
              "end": 63
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "startTime",
              "loc": {
                "start": 64,
                "end": 73
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 64,
              "end": 73
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "endTime",
              "loc": {
                "start": 74,
                "end": 81
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 74,
              "end": 81
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timezone",
              "loc": {
                "start": 82,
                "end": 90
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 82,
              "end": 90
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "exceptions",
              "loc": {
                "start": 91,
                "end": 101
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 108,
                      "end": 110
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 108,
                    "end": 110
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "originalStartTime",
                    "loc": {
                      "start": 115,
                      "end": 132
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 115,
                    "end": 132
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newStartTime",
                    "loc": {
                      "start": 137,
                      "end": 149
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 137,
                    "end": 149
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newEndTime",
                    "loc": {
                      "start": 154,
                      "end": 164
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 154,
                    "end": 164
                  }
                }
              ],
              "loc": {
                "start": 102,
                "end": 166
              }
            },
            "loc": {
              "start": 91,
              "end": 166
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "recurrences",
              "loc": {
                "start": 167,
                "end": 178
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 185,
                      "end": 187
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 185,
                    "end": 187
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "recurrenceType",
                    "loc": {
                      "start": 192,
                      "end": 206
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 192,
                    "end": 206
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "interval",
                    "loc": {
                      "start": 211,
                      "end": 219
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 211,
                    "end": 219
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfWeek",
                    "loc": {
                      "start": 224,
                      "end": 233
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 224,
                    "end": 233
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfMonth",
                    "loc": {
                      "start": 238,
                      "end": 248
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 238,
                    "end": 248
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "month",
                    "loc": {
                      "start": 253,
                      "end": 258
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 253,
                    "end": 258
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endDate",
                    "loc": {
                      "start": 263,
                      "end": 270
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 263,
                    "end": 270
                  }
                }
              ],
              "loc": {
                "start": 179,
                "end": 272
              }
            },
            "loc": {
              "start": 167,
              "end": 272
            }
          }
        ],
        "loc": {
          "start": 37,
          "end": 274
        }
      },
      "loc": {
        "start": 0,
        "end": 274
      }
    }
  },
  "rootValue": {},
  "operation": {
    "kind": "OperationDefinition",
    "operation": "mutation",
    "name": {
      "kind": "Name",
      "value": "emailLogIn",
      "loc": {
        "start": 285,
        "end": 295
      }
    },
    "variableDefinitions": [
      {
        "kind": "VariableDefinition",
        "variable": {
          "kind": "Variable",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 297,
              "end": 302
            }
          },
          "loc": {
            "start": 296,
            "end": 302
          }
        },
        "type": {
          "kind": "NonNullType",
          "type": {
            "kind": "NamedType",
            "name": {
              "kind": "Name",
              "value": "EmailLogInInput",
              "loc": {
                "start": 304,
                "end": 319
              }
            },
            "loc": {
              "start": 304,
              "end": 319
            }
          },
          "loc": {
            "start": 304,
            "end": 320
          }
        },
        "directives": [],
        "loc": {
          "start": 296,
          "end": 320
        }
      }
    ],
    "directives": [],
    "selectionSet": {
      "kind": "SelectionSet",
      "selections": [
        {
          "kind": "Field",
          "name": {
            "kind": "Name",
            "value": "emailLogIn",
            "loc": {
              "start": 326,
              "end": 336
            }
          },
          "arguments": [
            {
              "kind": "Argument",
              "name": {
                "kind": "Name",
                "value": "input",
                "loc": {
                  "start": 337,
                  "end": 342
                }
              },
              "value": {
                "kind": "Variable",
                "name": {
                  "kind": "Name",
                  "value": "input",
                  "loc": {
                    "start": 345,
                    "end": 350
                  }
                },
                "loc": {
                  "start": 344,
                  "end": 350
                }
              },
              "loc": {
                "start": 337,
                "end": 350
              }
            }
          ],
          "directives": [],
          "selectionSet": {
            "kind": "SelectionSet",
            "selections": [
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "isLoggedIn",
                  "loc": {
                    "start": 358,
                    "end": 368
                  }
                },
                "arguments": [],
                "directives": [],
                "loc": {
                  "start": 358,
                  "end": 368
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "timeZone",
                  "loc": {
                    "start": 373,
                    "end": 381
                  }
                },
                "arguments": [],
                "directives": [],
                "loc": {
                  "start": 373,
                  "end": 381
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "users",
                  "loc": {
                    "start": 386,
                    "end": 391
                  }
                },
                "arguments": [],
                "directives": [],
                "selectionSet": {
                  "kind": "SelectionSet",
                  "selections": [
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "activeFocusMode",
                        "loc": {
                          "start": 402,
                          "end": 417
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "selectionSet": {
                        "kind": "SelectionSet",
                        "selections": [
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "mode",
                              "loc": {
                                "start": 432,
                                "end": 436
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "filters",
                                    "loc": {
                                      "start": 455,
                                      "end": 462
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "selectionSet": {
                                    "kind": "SelectionSet",
                                    "selections": [
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "id",
                                          "loc": {
                                            "start": 485,
                                            "end": 487
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 485,
                                          "end": 487
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "filterType",
                                          "loc": {
                                            "start": 508,
                                            "end": 518
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 508,
                                          "end": 518
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "tag",
                                          "loc": {
                                            "start": 539,
                                            "end": 542
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "selectionSet": {
                                          "kind": "SelectionSet",
                                          "selections": [
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "id",
                                                "loc": {
                                                  "start": 569,
                                                  "end": 571
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 569,
                                                "end": 571
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "created_at",
                                                "loc": {
                                                  "start": 596,
                                                  "end": 606
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 596,
                                                "end": 606
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "tag",
                                                "loc": {
                                                  "start": 631,
                                                  "end": 634
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 631,
                                                "end": 634
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "bookmarks",
                                                "loc": {
                                                  "start": 659,
                                                  "end": 668
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 659,
                                                "end": 668
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "translations",
                                                "loc": {
                                                  "start": 693,
                                                  "end": 705
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "selectionSet": {
                                                "kind": "SelectionSet",
                                                "selections": [
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "id",
                                                      "loc": {
                                                        "start": 736,
                                                        "end": 738
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 736,
                                                      "end": 738
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "language",
                                                      "loc": {
                                                        "start": 767,
                                                        "end": 775
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 767,
                                                      "end": 775
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "description",
                                                      "loc": {
                                                        "start": 804,
                                                        "end": 815
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 804,
                                                      "end": 815
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 706,
                                                  "end": 841
                                                }
                                              },
                                              "loc": {
                                                "start": 693,
                                                "end": 841
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "you",
                                                "loc": {
                                                  "start": 866,
                                                  "end": 869
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "selectionSet": {
                                                "kind": "SelectionSet",
                                                "selections": [
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "isOwn",
                                                      "loc": {
                                                        "start": 900,
                                                        "end": 905
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 900,
                                                      "end": 905
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "isBookmarked",
                                                      "loc": {
                                                        "start": 934,
                                                        "end": 946
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 934,
                                                      "end": 946
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 870,
                                                  "end": 972
                                                }
                                              },
                                              "loc": {
                                                "start": 866,
                                                "end": 972
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 543,
                                            "end": 994
                                          }
                                        },
                                        "loc": {
                                          "start": 539,
                                          "end": 994
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "focusMode",
                                          "loc": {
                                            "start": 1015,
                                            "end": 1024
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "selectionSet": {
                                          "kind": "SelectionSet",
                                          "selections": [
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "labels",
                                                "loc": {
                                                  "start": 1051,
                                                  "end": 1057
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "selectionSet": {
                                                "kind": "SelectionSet",
                                                "selections": [
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "id",
                                                      "loc": {
                                                        "start": 1088,
                                                        "end": 1090
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 1088,
                                                      "end": 1090
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "color",
                                                      "loc": {
                                                        "start": 1119,
                                                        "end": 1124
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 1119,
                                                      "end": 1124
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "label",
                                                      "loc": {
                                                        "start": 1153,
                                                        "end": 1158
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 1153,
                                                      "end": 1158
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 1058,
                                                  "end": 1184
                                                }
                                              },
                                              "loc": {
                                                "start": 1051,
                                                "end": 1184
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "schedule",
                                                "loc": {
                                                  "start": 1209,
                                                  "end": 1217
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "selectionSet": {
                                                "kind": "SelectionSet",
                                                "selections": [
                                                  {
                                                    "kind": "FragmentSpread",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "Schedule_common",
                                                      "loc": {
                                                        "start": 1251,
                                                        "end": 1266
                                                      }
                                                    },
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 1248,
                                                      "end": 1266
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 1218,
                                                  "end": 1292
                                                }
                                              },
                                              "loc": {
                                                "start": 1209,
                                                "end": 1292
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "id",
                                                "loc": {
                                                  "start": 1317,
                                                  "end": 1319
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 1317,
                                                "end": 1319
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "name",
                                                "loc": {
                                                  "start": 1344,
                                                  "end": 1348
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 1344,
                                                "end": 1348
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "description",
                                                "loc": {
                                                  "start": 1373,
                                                  "end": 1384
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 1373,
                                                "end": 1384
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 1025,
                                            "end": 1406
                                          }
                                        },
                                        "loc": {
                                          "start": 1015,
                                          "end": 1406
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 463,
                                      "end": 1424
                                    }
                                  },
                                  "loc": {
                                    "start": 455,
                                    "end": 1424
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "labels",
                                    "loc": {
                                      "start": 1441,
                                      "end": 1447
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "selectionSet": {
                                    "kind": "SelectionSet",
                                    "selections": [
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "id",
                                          "loc": {
                                            "start": 1470,
                                            "end": 1472
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 1470,
                                          "end": 1472
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "color",
                                          "loc": {
                                            "start": 1493,
                                            "end": 1498
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 1493,
                                          "end": 1498
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "label",
                                          "loc": {
                                            "start": 1519,
                                            "end": 1524
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 1519,
                                          "end": 1524
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 1448,
                                      "end": 1542
                                    }
                                  },
                                  "loc": {
                                    "start": 1441,
                                    "end": 1542
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "reminderList",
                                    "loc": {
                                      "start": 1559,
                                      "end": 1571
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "selectionSet": {
                                    "kind": "SelectionSet",
                                    "selections": [
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "id",
                                          "loc": {
                                            "start": 1594,
                                            "end": 1596
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 1594,
                                          "end": 1596
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "created_at",
                                          "loc": {
                                            "start": 1617,
                                            "end": 1627
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 1617,
                                          "end": 1627
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "updated_at",
                                          "loc": {
                                            "start": 1648,
                                            "end": 1658
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 1648,
                                          "end": 1658
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "reminders",
                                          "loc": {
                                            "start": 1679,
                                            "end": 1688
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "selectionSet": {
                                          "kind": "SelectionSet",
                                          "selections": [
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "id",
                                                "loc": {
                                                  "start": 1715,
                                                  "end": 1717
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 1715,
                                                "end": 1717
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "created_at",
                                                "loc": {
                                                  "start": 1742,
                                                  "end": 1752
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 1742,
                                                "end": 1752
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "updated_at",
                                                "loc": {
                                                  "start": 1777,
                                                  "end": 1787
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 1777,
                                                "end": 1787
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "name",
                                                "loc": {
                                                  "start": 1812,
                                                  "end": 1816
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 1812,
                                                "end": 1816
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "description",
                                                "loc": {
                                                  "start": 1841,
                                                  "end": 1852
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 1841,
                                                "end": 1852
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "dueDate",
                                                "loc": {
                                                  "start": 1877,
                                                  "end": 1884
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 1877,
                                                "end": 1884
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "index",
                                                "loc": {
                                                  "start": 1909,
                                                  "end": 1914
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 1909,
                                                "end": 1914
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "isComplete",
                                                "loc": {
                                                  "start": 1939,
                                                  "end": 1949
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 1939,
                                                "end": 1949
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "reminderItems",
                                                "loc": {
                                                  "start": 1974,
                                                  "end": 1987
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "selectionSet": {
                                                "kind": "SelectionSet",
                                                "selections": [
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "id",
                                                      "loc": {
                                                        "start": 2018,
                                                        "end": 2020
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 2018,
                                                      "end": 2020
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "created_at",
                                                      "loc": {
                                                        "start": 2049,
                                                        "end": 2059
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 2049,
                                                      "end": 2059
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "updated_at",
                                                      "loc": {
                                                        "start": 2088,
                                                        "end": 2098
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 2088,
                                                      "end": 2098
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "name",
                                                      "loc": {
                                                        "start": 2127,
                                                        "end": 2131
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 2127,
                                                      "end": 2131
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "description",
                                                      "loc": {
                                                        "start": 2160,
                                                        "end": 2171
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 2160,
                                                      "end": 2171
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "dueDate",
                                                      "loc": {
                                                        "start": 2200,
                                                        "end": 2207
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 2200,
                                                      "end": 2207
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "index",
                                                      "loc": {
                                                        "start": 2236,
                                                        "end": 2241
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 2236,
                                                      "end": 2241
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "isComplete",
                                                      "loc": {
                                                        "start": 2270,
                                                        "end": 2280
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 2270,
                                                      "end": 2280
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 1988,
                                                  "end": 2306
                                                }
                                              },
                                              "loc": {
                                                "start": 1974,
                                                "end": 2306
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 1689,
                                            "end": 2328
                                          }
                                        },
                                        "loc": {
                                          "start": 1679,
                                          "end": 2328
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 1572,
                                      "end": 2346
                                    }
                                  },
                                  "loc": {
                                    "start": 1559,
                                    "end": 2346
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "resourceList",
                                    "loc": {
                                      "start": 2363,
                                      "end": 2375
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "selectionSet": {
                                    "kind": "SelectionSet",
                                    "selections": [
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "id",
                                          "loc": {
                                            "start": 2398,
                                            "end": 2400
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 2398,
                                          "end": 2400
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "created_at",
                                          "loc": {
                                            "start": 2421,
                                            "end": 2431
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 2421,
                                          "end": 2431
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "translations",
                                          "loc": {
                                            "start": 2452,
                                            "end": 2464
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "selectionSet": {
                                          "kind": "SelectionSet",
                                          "selections": [
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "id",
                                                "loc": {
                                                  "start": 2491,
                                                  "end": 2493
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 2491,
                                                "end": 2493
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "language",
                                                "loc": {
                                                  "start": 2518,
                                                  "end": 2526
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 2518,
                                                "end": 2526
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "description",
                                                "loc": {
                                                  "start": 2551,
                                                  "end": 2562
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 2551,
                                                "end": 2562
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "name",
                                                "loc": {
                                                  "start": 2587,
                                                  "end": 2591
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 2587,
                                                "end": 2591
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 2465,
                                            "end": 2613
                                          }
                                        },
                                        "loc": {
                                          "start": 2452,
                                          "end": 2613
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "resources",
                                          "loc": {
                                            "start": 2634,
                                            "end": 2643
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "selectionSet": {
                                          "kind": "SelectionSet",
                                          "selections": [
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "id",
                                                "loc": {
                                                  "start": 2670,
                                                  "end": 2672
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 2670,
                                                "end": 2672
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "index",
                                                "loc": {
                                                  "start": 2697,
                                                  "end": 2702
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 2697,
                                                "end": 2702
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "link",
                                                "loc": {
                                                  "start": 2727,
                                                  "end": 2731
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 2727,
                                                "end": 2731
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "usedFor",
                                                "loc": {
                                                  "start": 2756,
                                                  "end": 2763
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 2756,
                                                "end": 2763
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "translations",
                                                "loc": {
                                                  "start": 2788,
                                                  "end": 2800
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "selectionSet": {
                                                "kind": "SelectionSet",
                                                "selections": [
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "id",
                                                      "loc": {
                                                        "start": 2831,
                                                        "end": 2833
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 2831,
                                                      "end": 2833
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "language",
                                                      "loc": {
                                                        "start": 2862,
                                                        "end": 2870
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 2862,
                                                      "end": 2870
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "description",
                                                      "loc": {
                                                        "start": 2899,
                                                        "end": 2910
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 2899,
                                                      "end": 2910
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "name",
                                                      "loc": {
                                                        "start": 2939,
                                                        "end": 2943
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 2939,
                                                      "end": 2943
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 2801,
                                                  "end": 2969
                                                }
                                              },
                                              "loc": {
                                                "start": 2788,
                                                "end": 2969
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 2644,
                                            "end": 2991
                                          }
                                        },
                                        "loc": {
                                          "start": 2634,
                                          "end": 2991
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 2376,
                                      "end": 3009
                                    }
                                  },
                                  "loc": {
                                    "start": 2363,
                                    "end": 3009
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "schedule",
                                    "loc": {
                                      "start": 3026,
                                      "end": 3034
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "selectionSet": {
                                    "kind": "SelectionSet",
                                    "selections": [
                                      {
                                        "kind": "FragmentSpread",
                                        "name": {
                                          "kind": "Name",
                                          "value": "Schedule_common",
                                          "loc": {
                                            "start": 3060,
                                            "end": 3075
                                          }
                                        },
                                        "directives": [],
                                        "loc": {
                                          "start": 3057,
                                          "end": 3075
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 3035,
                                      "end": 3093
                                    }
                                  },
                                  "loc": {
                                    "start": 3026,
                                    "end": 3093
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "id",
                                    "loc": {
                                      "start": 3110,
                                      "end": 3112
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 3110,
                                    "end": 3112
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "name",
                                    "loc": {
                                      "start": 3129,
                                      "end": 3133
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 3129,
                                    "end": 3133
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "description",
                                    "loc": {
                                      "start": 3150,
                                      "end": 3161
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 3150,
                                    "end": 3161
                                  }
                                }
                              ],
                              "loc": {
                                "start": 437,
                                "end": 3175
                              }
                            },
                            "loc": {
                              "start": 432,
                              "end": 3175
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "stopCondition",
                              "loc": {
                                "start": 3188,
                                "end": 3201
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 3188,
                              "end": 3201
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "stopTime",
                              "loc": {
                                "start": 3214,
                                "end": 3222
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 3214,
                              "end": 3222
                            }
                          }
                        ],
                        "loc": {
                          "start": 418,
                          "end": 3232
                        }
                      },
                      "loc": {
                        "start": 402,
                        "end": 3232
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "apisCount",
                        "loc": {
                          "start": 3241,
                          "end": 3250
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 3241,
                        "end": 3250
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "bookmarkLists",
                        "loc": {
                          "start": 3259,
                          "end": 3272
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "selectionSet": {
                        "kind": "SelectionSet",
                        "selections": [
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "id",
                              "loc": {
                                "start": 3287,
                                "end": 3289
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 3287,
                              "end": 3289
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "created_at",
                              "loc": {
                                "start": 3302,
                                "end": 3312
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 3302,
                              "end": 3312
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "updated_at",
                              "loc": {
                                "start": 3325,
                                "end": 3335
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 3325,
                              "end": 3335
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "label",
                              "loc": {
                                "start": 3348,
                                "end": 3353
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 3348,
                              "end": 3353
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "bookmarksCount",
                              "loc": {
                                "start": 3366,
                                "end": 3380
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 3366,
                              "end": 3380
                            }
                          }
                        ],
                        "loc": {
                          "start": 3273,
                          "end": 3390
                        }
                      },
                      "loc": {
                        "start": 3259,
                        "end": 3390
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "focusModes",
                        "loc": {
                          "start": 3399,
                          "end": 3409
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "selectionSet": {
                        "kind": "SelectionSet",
                        "selections": [
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "filters",
                              "loc": {
                                "start": 3424,
                                "end": 3431
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "id",
                                    "loc": {
                                      "start": 3450,
                                      "end": 3452
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 3450,
                                    "end": 3452
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "filterType",
                                    "loc": {
                                      "start": 3469,
                                      "end": 3479
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 3469,
                                    "end": 3479
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "tag",
                                    "loc": {
                                      "start": 3496,
                                      "end": 3499
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "selectionSet": {
                                    "kind": "SelectionSet",
                                    "selections": [
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "id",
                                          "loc": {
                                            "start": 3522,
                                            "end": 3524
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 3522,
                                          "end": 3524
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "created_at",
                                          "loc": {
                                            "start": 3545,
                                            "end": 3555
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 3545,
                                          "end": 3555
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "tag",
                                          "loc": {
                                            "start": 3576,
                                            "end": 3579
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 3576,
                                          "end": 3579
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "bookmarks",
                                          "loc": {
                                            "start": 3600,
                                            "end": 3609
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 3600,
                                          "end": 3609
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "translations",
                                          "loc": {
                                            "start": 3630,
                                            "end": 3642
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "selectionSet": {
                                          "kind": "SelectionSet",
                                          "selections": [
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "id",
                                                "loc": {
                                                  "start": 3669,
                                                  "end": 3671
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3669,
                                                "end": 3671
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "language",
                                                "loc": {
                                                  "start": 3696,
                                                  "end": 3704
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3696,
                                                "end": 3704
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "description",
                                                "loc": {
                                                  "start": 3729,
                                                  "end": 3740
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3729,
                                                "end": 3740
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 3643,
                                            "end": 3762
                                          }
                                        },
                                        "loc": {
                                          "start": 3630,
                                          "end": 3762
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "you",
                                          "loc": {
                                            "start": 3783,
                                            "end": 3786
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "selectionSet": {
                                          "kind": "SelectionSet",
                                          "selections": [
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "isOwn",
                                                "loc": {
                                                  "start": 3813,
                                                  "end": 3818
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3813,
                                                "end": 3818
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "isBookmarked",
                                                "loc": {
                                                  "start": 3843,
                                                  "end": 3855
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3843,
                                                "end": 3855
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 3787,
                                            "end": 3877
                                          }
                                        },
                                        "loc": {
                                          "start": 3783,
                                          "end": 3877
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 3500,
                                      "end": 3895
                                    }
                                  },
                                  "loc": {
                                    "start": 3496,
                                    "end": 3895
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "focusMode",
                                    "loc": {
                                      "start": 3912,
                                      "end": 3921
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "selectionSet": {
                                    "kind": "SelectionSet",
                                    "selections": [
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "labels",
                                          "loc": {
                                            "start": 3944,
                                            "end": 3950
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "selectionSet": {
                                          "kind": "SelectionSet",
                                          "selections": [
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "id",
                                                "loc": {
                                                  "start": 3977,
                                                  "end": 3979
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3977,
                                                "end": 3979
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "color",
                                                "loc": {
                                                  "start": 4004,
                                                  "end": 4009
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4004,
                                                "end": 4009
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "label",
                                                "loc": {
                                                  "start": 4034,
                                                  "end": 4039
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4034,
                                                "end": 4039
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 3951,
                                            "end": 4061
                                          }
                                        },
                                        "loc": {
                                          "start": 3944,
                                          "end": 4061
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "schedule",
                                          "loc": {
                                            "start": 4082,
                                            "end": 4090
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "selectionSet": {
                                          "kind": "SelectionSet",
                                          "selections": [
                                            {
                                              "kind": "FragmentSpread",
                                              "name": {
                                                "kind": "Name",
                                                "value": "Schedule_common",
                                                "loc": {
                                                  "start": 4120,
                                                  "end": 4135
                                                }
                                              },
                                              "directives": [],
                                              "loc": {
                                                "start": 4117,
                                                "end": 4135
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 4091,
                                            "end": 4157
                                          }
                                        },
                                        "loc": {
                                          "start": 4082,
                                          "end": 4157
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "id",
                                          "loc": {
                                            "start": 4178,
                                            "end": 4180
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 4178,
                                          "end": 4180
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "name",
                                          "loc": {
                                            "start": 4201,
                                            "end": 4205
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 4201,
                                          "end": 4205
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "description",
                                          "loc": {
                                            "start": 4226,
                                            "end": 4237
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 4226,
                                          "end": 4237
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 3922,
                                      "end": 4255
                                    }
                                  },
                                  "loc": {
                                    "start": 3912,
                                    "end": 4255
                                  }
                                }
                              ],
                              "loc": {
                                "start": 3432,
                                "end": 4269
                              }
                            },
                            "loc": {
                              "start": 3424,
                              "end": 4269
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "labels",
                              "loc": {
                                "start": 4282,
                                "end": 4288
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "id",
                                    "loc": {
                                      "start": 4307,
                                      "end": 4309
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 4307,
                                    "end": 4309
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "color",
                                    "loc": {
                                      "start": 4326,
                                      "end": 4331
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 4326,
                                    "end": 4331
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "label",
                                    "loc": {
                                      "start": 4348,
                                      "end": 4353
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 4348,
                                    "end": 4353
                                  }
                                }
                              ],
                              "loc": {
                                "start": 4289,
                                "end": 4367
                              }
                            },
                            "loc": {
                              "start": 4282,
                              "end": 4367
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "reminderList",
                              "loc": {
                                "start": 4380,
                                "end": 4392
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "id",
                                    "loc": {
                                      "start": 4411,
                                      "end": 4413
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 4411,
                                    "end": 4413
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "created_at",
                                    "loc": {
                                      "start": 4430,
                                      "end": 4440
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 4430,
                                    "end": 4440
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "updated_at",
                                    "loc": {
                                      "start": 4457,
                                      "end": 4467
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 4457,
                                    "end": 4467
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "reminders",
                                    "loc": {
                                      "start": 4484,
                                      "end": 4493
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "selectionSet": {
                                    "kind": "SelectionSet",
                                    "selections": [
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "id",
                                          "loc": {
                                            "start": 4516,
                                            "end": 4518
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 4516,
                                          "end": 4518
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "created_at",
                                          "loc": {
                                            "start": 4539,
                                            "end": 4549
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 4539,
                                          "end": 4549
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "updated_at",
                                          "loc": {
                                            "start": 4570,
                                            "end": 4580
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 4570,
                                          "end": 4580
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "name",
                                          "loc": {
                                            "start": 4601,
                                            "end": 4605
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 4601,
                                          "end": 4605
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "description",
                                          "loc": {
                                            "start": 4626,
                                            "end": 4637
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 4626,
                                          "end": 4637
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "dueDate",
                                          "loc": {
                                            "start": 4658,
                                            "end": 4665
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 4658,
                                          "end": 4665
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "index",
                                          "loc": {
                                            "start": 4686,
                                            "end": 4691
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 4686,
                                          "end": 4691
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "isComplete",
                                          "loc": {
                                            "start": 4712,
                                            "end": 4722
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 4712,
                                          "end": 4722
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "reminderItems",
                                          "loc": {
                                            "start": 4743,
                                            "end": 4756
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "selectionSet": {
                                          "kind": "SelectionSet",
                                          "selections": [
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "id",
                                                "loc": {
                                                  "start": 4783,
                                                  "end": 4785
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4783,
                                                "end": 4785
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "created_at",
                                                "loc": {
                                                  "start": 4810,
                                                  "end": 4820
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4810,
                                                "end": 4820
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "updated_at",
                                                "loc": {
                                                  "start": 4845,
                                                  "end": 4855
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4845,
                                                "end": 4855
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "name",
                                                "loc": {
                                                  "start": 4880,
                                                  "end": 4884
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4880,
                                                "end": 4884
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "description",
                                                "loc": {
                                                  "start": 4909,
                                                  "end": 4920
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4909,
                                                "end": 4920
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "dueDate",
                                                "loc": {
                                                  "start": 4945,
                                                  "end": 4952
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4945,
                                                "end": 4952
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "index",
                                                "loc": {
                                                  "start": 4977,
                                                  "end": 4982
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4977,
                                                "end": 4982
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "isComplete",
                                                "loc": {
                                                  "start": 5007,
                                                  "end": 5017
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 5007,
                                                "end": 5017
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 4757,
                                            "end": 5039
                                          }
                                        },
                                        "loc": {
                                          "start": 4743,
                                          "end": 5039
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 4494,
                                      "end": 5057
                                    }
                                  },
                                  "loc": {
                                    "start": 4484,
                                    "end": 5057
                                  }
                                }
                              ],
                              "loc": {
                                "start": 4393,
                                "end": 5071
                              }
                            },
                            "loc": {
                              "start": 4380,
                              "end": 5071
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "resourceList",
                              "loc": {
                                "start": 5084,
                                "end": 5096
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "id",
                                    "loc": {
                                      "start": 5115,
                                      "end": 5117
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 5115,
                                    "end": 5117
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "created_at",
                                    "loc": {
                                      "start": 5134,
                                      "end": 5144
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 5134,
                                    "end": 5144
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "translations",
                                    "loc": {
                                      "start": 5161,
                                      "end": 5173
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "selectionSet": {
                                    "kind": "SelectionSet",
                                    "selections": [
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "id",
                                          "loc": {
                                            "start": 5196,
                                            "end": 5198
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 5196,
                                          "end": 5198
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "language",
                                          "loc": {
                                            "start": 5219,
                                            "end": 5227
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 5219,
                                          "end": 5227
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "description",
                                          "loc": {
                                            "start": 5248,
                                            "end": 5259
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 5248,
                                          "end": 5259
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "name",
                                          "loc": {
                                            "start": 5280,
                                            "end": 5284
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 5280,
                                          "end": 5284
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 5174,
                                      "end": 5302
                                    }
                                  },
                                  "loc": {
                                    "start": 5161,
                                    "end": 5302
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "resources",
                                    "loc": {
                                      "start": 5319,
                                      "end": 5328
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "selectionSet": {
                                    "kind": "SelectionSet",
                                    "selections": [
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "id",
                                          "loc": {
                                            "start": 5351,
                                            "end": 5353
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 5351,
                                          "end": 5353
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "index",
                                          "loc": {
                                            "start": 5374,
                                            "end": 5379
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 5374,
                                          "end": 5379
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "link",
                                          "loc": {
                                            "start": 5400,
                                            "end": 5404
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 5400,
                                          "end": 5404
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "usedFor",
                                          "loc": {
                                            "start": 5425,
                                            "end": 5432
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 5425,
                                          "end": 5432
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "translations",
                                          "loc": {
                                            "start": 5453,
                                            "end": 5465
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "selectionSet": {
                                          "kind": "SelectionSet",
                                          "selections": [
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "id",
                                                "loc": {
                                                  "start": 5492,
                                                  "end": 5494
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 5492,
                                                "end": 5494
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "language",
                                                "loc": {
                                                  "start": 5519,
                                                  "end": 5527
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 5519,
                                                "end": 5527
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "description",
                                                "loc": {
                                                  "start": 5552,
                                                  "end": 5563
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 5552,
                                                "end": 5563
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "name",
                                                "loc": {
                                                  "start": 5588,
                                                  "end": 5592
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 5588,
                                                "end": 5592
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 5466,
                                            "end": 5614
                                          }
                                        },
                                        "loc": {
                                          "start": 5453,
                                          "end": 5614
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 5329,
                                      "end": 5632
                                    }
                                  },
                                  "loc": {
                                    "start": 5319,
                                    "end": 5632
                                  }
                                }
                              ],
                              "loc": {
                                "start": 5097,
                                "end": 5646
                              }
                            },
                            "loc": {
                              "start": 5084,
                              "end": 5646
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "schedule",
                              "loc": {
                                "start": 5659,
                                "end": 5667
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "FragmentSpread",
                                  "name": {
                                    "kind": "Name",
                                    "value": "Schedule_common",
                                    "loc": {
                                      "start": 5689,
                                      "end": 5704
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 5686,
                                    "end": 5704
                                  }
                                }
                              ],
                              "loc": {
                                "start": 5668,
                                "end": 5718
                              }
                            },
                            "loc": {
                              "start": 5659,
                              "end": 5718
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "id",
                              "loc": {
                                "start": 5731,
                                "end": 5733
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 5731,
                              "end": 5733
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "name",
                              "loc": {
                                "start": 5746,
                                "end": 5750
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 5746,
                              "end": 5750
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "description",
                              "loc": {
                                "start": 5763,
                                "end": 5774
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 5763,
                              "end": 5774
                            }
                          }
                        ],
                        "loc": {
                          "start": 3410,
                          "end": 5784
                        }
                      },
                      "loc": {
                        "start": 3399,
                        "end": 5784
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "handle",
                        "loc": {
                          "start": 5793,
                          "end": 5799
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 5793,
                        "end": 5799
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "hasPremium",
                        "loc": {
                          "start": 5808,
                          "end": 5818
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 5808,
                        "end": 5818
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "id",
                        "loc": {
                          "start": 5827,
                          "end": 5829
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 5827,
                        "end": 5829
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "languages",
                        "loc": {
                          "start": 5838,
                          "end": 5847
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 5838,
                        "end": 5847
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "membershipsCount",
                        "loc": {
                          "start": 5856,
                          "end": 5872
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 5856,
                        "end": 5872
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "name",
                        "loc": {
                          "start": 5881,
                          "end": 5885
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 5881,
                        "end": 5885
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "notesCount",
                        "loc": {
                          "start": 5894,
                          "end": 5904
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 5894,
                        "end": 5904
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "projectsCount",
                        "loc": {
                          "start": 5913,
                          "end": 5926
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 5913,
                        "end": 5926
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "questionsAskedCount",
                        "loc": {
                          "start": 5935,
                          "end": 5954
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 5935,
                        "end": 5954
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "routinesCount",
                        "loc": {
                          "start": 5963,
                          "end": 5976
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 5963,
                        "end": 5976
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "smartContractsCount",
                        "loc": {
                          "start": 5985,
                          "end": 6004
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 5985,
                        "end": 6004
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "standardsCount",
                        "loc": {
                          "start": 6013,
                          "end": 6027
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 6013,
                        "end": 6027
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "theme",
                        "loc": {
                          "start": 6036,
                          "end": 6041
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 6036,
                        "end": 6041
                      }
                    }
                  ],
                  "loc": {
                    "start": 392,
                    "end": 6047
                  }
                },
                "loc": {
                  "start": 386,
                  "end": 6047
                }
              }
            ],
            "loc": {
              "start": 352,
              "end": 6051
            }
          },
          "loc": {
            "start": 326,
            "end": 6051
          }
        }
      ],
      "loc": {
        "start": 322,
        "end": 6053
      }
    },
    "loc": {
      "start": 276,
      "end": 6053
    }
  },
  "variableValues": {},
  "path": {
    "key": "auth_emailLogIn"
  }
} as const;
