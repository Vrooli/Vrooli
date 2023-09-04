export const auth_emailSignUp = {
  "fieldName": "emailSignUp",
  "fieldNodes": [
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "emailSignUp",
        "loc": {
          "start": 329,
          "end": 340
        }
      },
      "arguments": [
        {
          "kind": "Argument",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 341,
              "end": 346
            }
          },
          "value": {
            "kind": "Variable",
            "name": {
              "kind": "Name",
              "value": "input",
              "loc": {
                "start": 349,
                "end": 354
              }
            },
            "loc": {
              "start": 348,
              "end": 354
            }
          },
          "loc": {
            "start": 341,
            "end": 354
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
                "start": 362,
                "end": 372
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 362,
              "end": 372
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timeZone",
              "loc": {
                "start": 377,
                "end": 385
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 377,
              "end": 385
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "users",
              "loc": {
                "start": 390,
                "end": 395
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
                      "start": 406,
                      "end": 421
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
                            "start": 436,
                            "end": 440
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
                                  "start": 459,
                                  "end": 466
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
                                        "start": 489,
                                        "end": 491
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 489,
                                      "end": 491
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "filterType",
                                      "loc": {
                                        "start": 512,
                                        "end": 522
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 512,
                                      "end": 522
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "tag",
                                      "loc": {
                                        "start": 543,
                                        "end": 546
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
                                              "start": 573,
                                              "end": 575
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 573,
                                            "end": 575
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 600,
                                              "end": 610
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 600,
                                            "end": 610
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "tag",
                                            "loc": {
                                              "start": 635,
                                              "end": 638
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 635,
                                            "end": 638
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "bookmarks",
                                            "loc": {
                                              "start": 663,
                                              "end": 672
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 663,
                                            "end": 672
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "translations",
                                            "loc": {
                                              "start": 697,
                                              "end": 709
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
                                                    "start": 740,
                                                    "end": 742
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 740,
                                                  "end": 742
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "language",
                                                  "loc": {
                                                    "start": 771,
                                                    "end": 779
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 771,
                                                  "end": 779
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "description",
                                                  "loc": {
                                                    "start": 808,
                                                    "end": 819
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 808,
                                                  "end": 819
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 710,
                                              "end": 845
                                            }
                                          },
                                          "loc": {
                                            "start": 697,
                                            "end": 845
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "you",
                                            "loc": {
                                              "start": 870,
                                              "end": 873
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
                                                    "start": 904,
                                                    "end": 909
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 904,
                                                  "end": 909
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isBookmarked",
                                                  "loc": {
                                                    "start": 938,
                                                    "end": 950
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 938,
                                                  "end": 950
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 874,
                                              "end": 976
                                            }
                                          },
                                          "loc": {
                                            "start": 870,
                                            "end": 976
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 547,
                                        "end": 998
                                      }
                                    },
                                    "loc": {
                                      "start": 543,
                                      "end": 998
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "focusMode",
                                      "loc": {
                                        "start": 1019,
                                        "end": 1028
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
                                              "start": 1055,
                                              "end": 1061
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
                                                    "start": 1092,
                                                    "end": 1094
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1092,
                                                  "end": 1094
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "color",
                                                  "loc": {
                                                    "start": 1123,
                                                    "end": 1128
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1123,
                                                  "end": 1128
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "label",
                                                  "loc": {
                                                    "start": 1157,
                                                    "end": 1162
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1157,
                                                  "end": 1162
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 1062,
                                              "end": 1188
                                            }
                                          },
                                          "loc": {
                                            "start": 1055,
                                            "end": 1188
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "reminderList",
                                            "loc": {
                                              "start": 1213,
                                              "end": 1225
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
                                                    "start": 1256,
                                                    "end": 1258
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1256,
                                                  "end": 1258
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "created_at",
                                                  "loc": {
                                                    "start": 1287,
                                                    "end": 1297
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1287,
                                                  "end": 1297
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "updated_at",
                                                  "loc": {
                                                    "start": 1326,
                                                    "end": 1336
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1326,
                                                  "end": 1336
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "reminders",
                                                  "loc": {
                                                    "start": 1365,
                                                    "end": 1374
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
                                                          "start": 1409,
                                                          "end": 1411
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1409,
                                                        "end": 1411
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "created_at",
                                                        "loc": {
                                                          "start": 1444,
                                                          "end": 1454
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1444,
                                                        "end": 1454
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "updated_at",
                                                        "loc": {
                                                          "start": 1487,
                                                          "end": 1497
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1487,
                                                        "end": 1497
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "name",
                                                        "loc": {
                                                          "start": 1530,
                                                          "end": 1534
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1530,
                                                        "end": 1534
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "description",
                                                        "loc": {
                                                          "start": 1567,
                                                          "end": 1578
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1567,
                                                        "end": 1578
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "dueDate",
                                                        "loc": {
                                                          "start": 1611,
                                                          "end": 1618
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1611,
                                                        "end": 1618
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "index",
                                                        "loc": {
                                                          "start": 1651,
                                                          "end": 1656
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1651,
                                                        "end": 1656
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "isComplete",
                                                        "loc": {
                                                          "start": 1689,
                                                          "end": 1699
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1689,
                                                        "end": 1699
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "reminderItems",
                                                        "loc": {
                                                          "start": 1732,
                                                          "end": 1745
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
                                                                "start": 1784,
                                                                "end": 1786
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 1784,
                                                              "end": 1786
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "created_at",
                                                              "loc": {
                                                                "start": 1823,
                                                                "end": 1833
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 1823,
                                                              "end": 1833
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "updated_at",
                                                              "loc": {
                                                                "start": 1870,
                                                                "end": 1880
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 1870,
                                                              "end": 1880
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "name",
                                                              "loc": {
                                                                "start": 1917,
                                                                "end": 1921
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 1917,
                                                              "end": 1921
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "description",
                                                              "loc": {
                                                                "start": 1958,
                                                                "end": 1969
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 1958,
                                                              "end": 1969
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "dueDate",
                                                              "loc": {
                                                                "start": 2006,
                                                                "end": 2013
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 2006,
                                                              "end": 2013
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "index",
                                                              "loc": {
                                                                "start": 2050,
                                                                "end": 2055
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 2050,
                                                              "end": 2055
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "isComplete",
                                                              "loc": {
                                                                "start": 2092,
                                                                "end": 2102
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 2092,
                                                              "end": 2102
                                                            }
                                                          }
                                                        ],
                                                        "loc": {
                                                          "start": 1746,
                                                          "end": 2136
                                                        }
                                                      },
                                                      "loc": {
                                                        "start": 1732,
                                                        "end": 2136
                                                      }
                                                    }
                                                  ],
                                                  "loc": {
                                                    "start": 1375,
                                                    "end": 2166
                                                  }
                                                },
                                                "loc": {
                                                  "start": 1365,
                                                  "end": 2166
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 1226,
                                              "end": 2192
                                            }
                                          },
                                          "loc": {
                                            "start": 1213,
                                            "end": 2192
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "resourceList",
                                            "loc": {
                                              "start": 2217,
                                              "end": 2229
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
                                                    "start": 2260,
                                                    "end": 2262
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2260,
                                                  "end": 2262
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "created_at",
                                                  "loc": {
                                                    "start": 2291,
                                                    "end": 2301
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2291,
                                                  "end": 2301
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "translations",
                                                  "loc": {
                                                    "start": 2330,
                                                    "end": 2342
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
                                                          "start": 2377,
                                                          "end": 2379
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 2377,
                                                        "end": 2379
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "language",
                                                        "loc": {
                                                          "start": 2412,
                                                          "end": 2420
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 2412,
                                                        "end": 2420
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "description",
                                                        "loc": {
                                                          "start": 2453,
                                                          "end": 2464
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 2453,
                                                        "end": 2464
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "name",
                                                        "loc": {
                                                          "start": 2497,
                                                          "end": 2501
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 2497,
                                                        "end": 2501
                                                      }
                                                    }
                                                  ],
                                                  "loc": {
                                                    "start": 2343,
                                                    "end": 2531
                                                  }
                                                },
                                                "loc": {
                                                  "start": 2330,
                                                  "end": 2531
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "resources",
                                                  "loc": {
                                                    "start": 2560,
                                                    "end": 2569
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
                                                          "start": 2604,
                                                          "end": 2606
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 2604,
                                                        "end": 2606
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "index",
                                                        "loc": {
                                                          "start": 2639,
                                                          "end": 2644
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 2639,
                                                        "end": 2644
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "link",
                                                        "loc": {
                                                          "start": 2677,
                                                          "end": 2681
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 2677,
                                                        "end": 2681
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "usedFor",
                                                        "loc": {
                                                          "start": 2714,
                                                          "end": 2721
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 2714,
                                                        "end": 2721
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "translations",
                                                        "loc": {
                                                          "start": 2754,
                                                          "end": 2766
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
                                                                "start": 2805,
                                                                "end": 2807
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 2805,
                                                              "end": 2807
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "language",
                                                              "loc": {
                                                                "start": 2844,
                                                                "end": 2852
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 2844,
                                                              "end": 2852
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "description",
                                                              "loc": {
                                                                "start": 2889,
                                                                "end": 2900
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 2889,
                                                              "end": 2900
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "name",
                                                              "loc": {
                                                                "start": 2937,
                                                                "end": 2941
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 2937,
                                                              "end": 2941
                                                            }
                                                          }
                                                        ],
                                                        "loc": {
                                                          "start": 2767,
                                                          "end": 2975
                                                        }
                                                      },
                                                      "loc": {
                                                        "start": 2754,
                                                        "end": 2975
                                                      }
                                                    }
                                                  ],
                                                  "loc": {
                                                    "start": 2570,
                                                    "end": 3005
                                                  }
                                                },
                                                "loc": {
                                                  "start": 2560,
                                                  "end": 3005
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 2230,
                                              "end": 3031
                                            }
                                          },
                                          "loc": {
                                            "start": 2217,
                                            "end": 3031
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "schedule",
                                            "loc": {
                                              "start": 3056,
                                              "end": 3064
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
                                                    "start": 3098,
                                                    "end": 3113
                                                  }
                                                },
                                                "directives": [],
                                                "loc": {
                                                  "start": 3095,
                                                  "end": 3113
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 3065,
                                              "end": 3139
                                            }
                                          },
                                          "loc": {
                                            "start": 3056,
                                            "end": 3139
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 3164,
                                              "end": 3166
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3164,
                                            "end": 3166
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 3191,
                                              "end": 3195
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3191,
                                            "end": 3195
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 3220,
                                              "end": 3231
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3220,
                                            "end": 3231
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 1029,
                                        "end": 3253
                                      }
                                    },
                                    "loc": {
                                      "start": 1019,
                                      "end": 3253
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 467,
                                  "end": 3271
                                }
                              },
                              "loc": {
                                "start": 459,
                                "end": 3271
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "labels",
                                "loc": {
                                  "start": 3288,
                                  "end": 3294
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
                                        "start": 3317,
                                        "end": 3319
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3317,
                                      "end": 3319
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "color",
                                      "loc": {
                                        "start": 3340,
                                        "end": 3345
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3340,
                                      "end": 3345
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "label",
                                      "loc": {
                                        "start": 3366,
                                        "end": 3371
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3366,
                                      "end": 3371
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 3295,
                                  "end": 3389
                                }
                              },
                              "loc": {
                                "start": 3288,
                                "end": 3389
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "reminderList",
                                "loc": {
                                  "start": 3406,
                                  "end": 3418
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
                                        "start": 3441,
                                        "end": 3443
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3441,
                                      "end": 3443
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 3464,
                                        "end": 3474
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3464,
                                      "end": 3474
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 3495,
                                        "end": 3505
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3495,
                                      "end": 3505
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "reminders",
                                      "loc": {
                                        "start": 3526,
                                        "end": 3535
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
                                              "start": 3562,
                                              "end": 3564
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3562,
                                            "end": 3564
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 3589,
                                              "end": 3599
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3589,
                                            "end": 3599
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "updated_at",
                                            "loc": {
                                              "start": 3624,
                                              "end": 3634
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3624,
                                            "end": 3634
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 3659,
                                              "end": 3663
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3659,
                                            "end": 3663
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 3688,
                                              "end": 3699
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3688,
                                            "end": 3699
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "dueDate",
                                            "loc": {
                                              "start": 3724,
                                              "end": 3731
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3724,
                                            "end": 3731
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "index",
                                            "loc": {
                                              "start": 3756,
                                              "end": 3761
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3756,
                                            "end": 3761
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isComplete",
                                            "loc": {
                                              "start": 3786,
                                              "end": 3796
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3786,
                                            "end": 3796
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "reminderItems",
                                            "loc": {
                                              "start": 3821,
                                              "end": 3834
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
                                                    "start": 3865,
                                                    "end": 3867
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3865,
                                                  "end": 3867
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "created_at",
                                                  "loc": {
                                                    "start": 3896,
                                                    "end": 3906
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3896,
                                                  "end": 3906
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "updated_at",
                                                  "loc": {
                                                    "start": 3935,
                                                    "end": 3945
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3935,
                                                  "end": 3945
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "name",
                                                  "loc": {
                                                    "start": 3974,
                                                    "end": 3978
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3974,
                                                  "end": 3978
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "description",
                                                  "loc": {
                                                    "start": 4007,
                                                    "end": 4018
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4007,
                                                  "end": 4018
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "dueDate",
                                                  "loc": {
                                                    "start": 4047,
                                                    "end": 4054
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4047,
                                                  "end": 4054
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "index",
                                                  "loc": {
                                                    "start": 4083,
                                                    "end": 4088
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4083,
                                                  "end": 4088
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isComplete",
                                                  "loc": {
                                                    "start": 4117,
                                                    "end": 4127
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4117,
                                                  "end": 4127
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 3835,
                                              "end": 4153
                                            }
                                          },
                                          "loc": {
                                            "start": 3821,
                                            "end": 4153
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 3536,
                                        "end": 4175
                                      }
                                    },
                                    "loc": {
                                      "start": 3526,
                                      "end": 4175
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 3419,
                                  "end": 4193
                                }
                              },
                              "loc": {
                                "start": 3406,
                                "end": 4193
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "resourceList",
                                "loc": {
                                  "start": 4210,
                                  "end": 4222
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
                                        "start": 4245,
                                        "end": 4247
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4245,
                                      "end": 4247
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 4268,
                                        "end": 4278
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4268,
                                      "end": 4278
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "translations",
                                      "loc": {
                                        "start": 4299,
                                        "end": 4311
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
                                              "start": 4338,
                                              "end": 4340
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4338,
                                            "end": 4340
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "language",
                                            "loc": {
                                              "start": 4365,
                                              "end": 4373
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4365,
                                            "end": 4373
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 4398,
                                              "end": 4409
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4398,
                                            "end": 4409
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 4434,
                                              "end": 4438
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4434,
                                            "end": 4438
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 4312,
                                        "end": 4460
                                      }
                                    },
                                    "loc": {
                                      "start": 4299,
                                      "end": 4460
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "resources",
                                      "loc": {
                                        "start": 4481,
                                        "end": 4490
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
                                              "start": 4517,
                                              "end": 4519
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4517,
                                            "end": 4519
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "index",
                                            "loc": {
                                              "start": 4544,
                                              "end": 4549
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4544,
                                            "end": 4549
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "link",
                                            "loc": {
                                              "start": 4574,
                                              "end": 4578
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4574,
                                            "end": 4578
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "usedFor",
                                            "loc": {
                                              "start": 4603,
                                              "end": 4610
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4603,
                                            "end": 4610
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "translations",
                                            "loc": {
                                              "start": 4635,
                                              "end": 4647
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
                                                    "start": 4678,
                                                    "end": 4680
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4678,
                                                  "end": 4680
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "language",
                                                  "loc": {
                                                    "start": 4709,
                                                    "end": 4717
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4709,
                                                  "end": 4717
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "description",
                                                  "loc": {
                                                    "start": 4746,
                                                    "end": 4757
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4746,
                                                  "end": 4757
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "name",
                                                  "loc": {
                                                    "start": 4786,
                                                    "end": 4790
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4786,
                                                  "end": 4790
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 4648,
                                              "end": 4816
                                            }
                                          },
                                          "loc": {
                                            "start": 4635,
                                            "end": 4816
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 4491,
                                        "end": 4838
                                      }
                                    },
                                    "loc": {
                                      "start": 4481,
                                      "end": 4838
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 4223,
                                  "end": 4856
                                }
                              },
                              "loc": {
                                "start": 4210,
                                "end": 4856
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "schedule",
                                "loc": {
                                  "start": 4873,
                                  "end": 4881
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
                                        "start": 4907,
                                        "end": 4922
                                      }
                                    },
                                    "directives": [],
                                    "loc": {
                                      "start": 4904,
                                      "end": 4922
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 4882,
                                  "end": 4940
                                }
                              },
                              "loc": {
                                "start": 4873,
                                "end": 4940
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 4957,
                                  "end": 4959
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4957,
                                "end": 4959
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 4976,
                                  "end": 4980
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4976,
                                "end": 4980
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 4997,
                                  "end": 5008
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4997,
                                "end": 5008
                              }
                            }
                          ],
                          "loc": {
                            "start": 441,
                            "end": 5022
                          }
                        },
                        "loc": {
                          "start": 436,
                          "end": 5022
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "stopCondition",
                          "loc": {
                            "start": 5035,
                            "end": 5048
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5035,
                          "end": 5048
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "stopTime",
                          "loc": {
                            "start": 5061,
                            "end": 5069
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5061,
                          "end": 5069
                        }
                      }
                    ],
                    "loc": {
                      "start": 422,
                      "end": 5079
                    }
                  },
                  "loc": {
                    "start": 406,
                    "end": 5079
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "apisCount",
                    "loc": {
                      "start": 5088,
                      "end": 5097
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5088,
                    "end": 5097
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "bookmarkLists",
                    "loc": {
                      "start": 5106,
                      "end": 5119
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
                            "start": 5134,
                            "end": 5136
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5134,
                          "end": 5136
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 5149,
                            "end": 5159
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5149,
                          "end": 5159
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 5172,
                            "end": 5182
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5172,
                          "end": 5182
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "label",
                          "loc": {
                            "start": 5195,
                            "end": 5200
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5195,
                          "end": 5200
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "bookmarksCount",
                          "loc": {
                            "start": 5213,
                            "end": 5227
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5213,
                          "end": 5227
                        }
                      }
                    ],
                    "loc": {
                      "start": 5120,
                      "end": 5237
                    }
                  },
                  "loc": {
                    "start": 5106,
                    "end": 5237
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "focusModes",
                    "loc": {
                      "start": 5246,
                      "end": 5256
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
                            "start": 5271,
                            "end": 5278
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
                                  "start": 5297,
                                  "end": 5299
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5297,
                                "end": 5299
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "filterType",
                                "loc": {
                                  "start": 5316,
                                  "end": 5326
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5316,
                                "end": 5326
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "tag",
                                "loc": {
                                  "start": 5343,
                                  "end": 5346
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
                                        "start": 5369,
                                        "end": 5371
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5369,
                                      "end": 5371
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 5392,
                                        "end": 5402
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5392,
                                      "end": 5402
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "tag",
                                      "loc": {
                                        "start": 5423,
                                        "end": 5426
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5423,
                                      "end": 5426
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "bookmarks",
                                      "loc": {
                                        "start": 5447,
                                        "end": 5456
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5447,
                                      "end": 5456
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "translations",
                                      "loc": {
                                        "start": 5477,
                                        "end": 5489
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
                                              "start": 5516,
                                              "end": 5518
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5516,
                                            "end": 5518
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "language",
                                            "loc": {
                                              "start": 5543,
                                              "end": 5551
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5543,
                                            "end": 5551
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 5576,
                                              "end": 5587
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5576,
                                            "end": 5587
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 5490,
                                        "end": 5609
                                      }
                                    },
                                    "loc": {
                                      "start": 5477,
                                      "end": 5609
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "you",
                                      "loc": {
                                        "start": 5630,
                                        "end": 5633
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
                                              "start": 5660,
                                              "end": 5665
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5660,
                                            "end": 5665
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isBookmarked",
                                            "loc": {
                                              "start": 5690,
                                              "end": 5702
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5690,
                                            "end": 5702
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 5634,
                                        "end": 5724
                                      }
                                    },
                                    "loc": {
                                      "start": 5630,
                                      "end": 5724
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 5347,
                                  "end": 5742
                                }
                              },
                              "loc": {
                                "start": 5343,
                                "end": 5742
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "focusMode",
                                "loc": {
                                  "start": 5759,
                                  "end": 5768
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
                                        "start": 5791,
                                        "end": 5797
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
                                              "start": 5824,
                                              "end": 5826
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5824,
                                            "end": 5826
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "color",
                                            "loc": {
                                              "start": 5851,
                                              "end": 5856
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5851,
                                            "end": 5856
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "label",
                                            "loc": {
                                              "start": 5881,
                                              "end": 5886
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5881,
                                            "end": 5886
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 5798,
                                        "end": 5908
                                      }
                                    },
                                    "loc": {
                                      "start": 5791,
                                      "end": 5908
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "reminderList",
                                      "loc": {
                                        "start": 5929,
                                        "end": 5941
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
                                              "start": 5968,
                                              "end": 5970
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5968,
                                            "end": 5970
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 5995,
                                              "end": 6005
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5995,
                                            "end": 6005
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "updated_at",
                                            "loc": {
                                              "start": 6030,
                                              "end": 6040
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 6030,
                                            "end": 6040
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "reminders",
                                            "loc": {
                                              "start": 6065,
                                              "end": 6074
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
                                                    "start": 6105,
                                                    "end": 6107
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6105,
                                                  "end": 6107
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "created_at",
                                                  "loc": {
                                                    "start": 6136,
                                                    "end": 6146
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6136,
                                                  "end": 6146
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "updated_at",
                                                  "loc": {
                                                    "start": 6175,
                                                    "end": 6185
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6175,
                                                  "end": 6185
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "name",
                                                  "loc": {
                                                    "start": 6214,
                                                    "end": 6218
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6214,
                                                  "end": 6218
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "description",
                                                  "loc": {
                                                    "start": 6247,
                                                    "end": 6258
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6247,
                                                  "end": 6258
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "dueDate",
                                                  "loc": {
                                                    "start": 6287,
                                                    "end": 6294
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6287,
                                                  "end": 6294
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "index",
                                                  "loc": {
                                                    "start": 6323,
                                                    "end": 6328
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6323,
                                                  "end": 6328
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isComplete",
                                                  "loc": {
                                                    "start": 6357,
                                                    "end": 6367
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6357,
                                                  "end": 6367
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "reminderItems",
                                                  "loc": {
                                                    "start": 6396,
                                                    "end": 6409
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
                                                          "start": 6444,
                                                          "end": 6446
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 6444,
                                                        "end": 6446
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "created_at",
                                                        "loc": {
                                                          "start": 6479,
                                                          "end": 6489
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 6479,
                                                        "end": 6489
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "updated_at",
                                                        "loc": {
                                                          "start": 6522,
                                                          "end": 6532
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 6522,
                                                        "end": 6532
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "name",
                                                        "loc": {
                                                          "start": 6565,
                                                          "end": 6569
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 6565,
                                                        "end": 6569
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "description",
                                                        "loc": {
                                                          "start": 6602,
                                                          "end": 6613
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 6602,
                                                        "end": 6613
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "dueDate",
                                                        "loc": {
                                                          "start": 6646,
                                                          "end": 6653
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 6646,
                                                        "end": 6653
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "index",
                                                        "loc": {
                                                          "start": 6686,
                                                          "end": 6691
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 6686,
                                                        "end": 6691
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "isComplete",
                                                        "loc": {
                                                          "start": 6724,
                                                          "end": 6734
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 6724,
                                                        "end": 6734
                                                      }
                                                    }
                                                  ],
                                                  "loc": {
                                                    "start": 6410,
                                                    "end": 6764
                                                  }
                                                },
                                                "loc": {
                                                  "start": 6396,
                                                  "end": 6764
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 6075,
                                              "end": 6790
                                            }
                                          },
                                          "loc": {
                                            "start": 6065,
                                            "end": 6790
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 5942,
                                        "end": 6812
                                      }
                                    },
                                    "loc": {
                                      "start": 5929,
                                      "end": 6812
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "resourceList",
                                      "loc": {
                                        "start": 6833,
                                        "end": 6845
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
                                              "start": 6872,
                                              "end": 6874
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 6872,
                                            "end": 6874
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 6899,
                                              "end": 6909
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 6899,
                                            "end": 6909
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "translations",
                                            "loc": {
                                              "start": 6934,
                                              "end": 6946
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
                                                    "start": 6977,
                                                    "end": 6979
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6977,
                                                  "end": 6979
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "language",
                                                  "loc": {
                                                    "start": 7008,
                                                    "end": 7016
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 7008,
                                                  "end": 7016
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "description",
                                                  "loc": {
                                                    "start": 7045,
                                                    "end": 7056
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 7045,
                                                  "end": 7056
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "name",
                                                  "loc": {
                                                    "start": 7085,
                                                    "end": 7089
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 7085,
                                                  "end": 7089
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 6947,
                                              "end": 7115
                                            }
                                          },
                                          "loc": {
                                            "start": 6934,
                                            "end": 7115
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "resources",
                                            "loc": {
                                              "start": 7140,
                                              "end": 7149
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
                                                    "start": 7180,
                                                    "end": 7182
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 7180,
                                                  "end": 7182
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "index",
                                                  "loc": {
                                                    "start": 7211,
                                                    "end": 7216
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 7211,
                                                  "end": 7216
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "link",
                                                  "loc": {
                                                    "start": 7245,
                                                    "end": 7249
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 7245,
                                                  "end": 7249
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "usedFor",
                                                  "loc": {
                                                    "start": 7278,
                                                    "end": 7285
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 7278,
                                                  "end": 7285
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "translations",
                                                  "loc": {
                                                    "start": 7314,
                                                    "end": 7326
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
                                                          "start": 7361,
                                                          "end": 7363
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 7361,
                                                        "end": 7363
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "language",
                                                        "loc": {
                                                          "start": 7396,
                                                          "end": 7404
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 7396,
                                                        "end": 7404
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "description",
                                                        "loc": {
                                                          "start": 7437,
                                                          "end": 7448
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 7437,
                                                        "end": 7448
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "name",
                                                        "loc": {
                                                          "start": 7481,
                                                          "end": 7485
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 7481,
                                                        "end": 7485
                                                      }
                                                    }
                                                  ],
                                                  "loc": {
                                                    "start": 7327,
                                                    "end": 7515
                                                  }
                                                },
                                                "loc": {
                                                  "start": 7314,
                                                  "end": 7515
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 7150,
                                              "end": 7541
                                            }
                                          },
                                          "loc": {
                                            "start": 7140,
                                            "end": 7541
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 6846,
                                        "end": 7563
                                      }
                                    },
                                    "loc": {
                                      "start": 6833,
                                      "end": 7563
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "schedule",
                                      "loc": {
                                        "start": 7584,
                                        "end": 7592
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
                                              "start": 7622,
                                              "end": 7637
                                            }
                                          },
                                          "directives": [],
                                          "loc": {
                                            "start": 7619,
                                            "end": 7637
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 7593,
                                        "end": 7659
                                      }
                                    },
                                    "loc": {
                                      "start": 7584,
                                      "end": 7659
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 7680,
                                        "end": 7682
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 7680,
                                      "end": 7682
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 7703,
                                        "end": 7707
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 7703,
                                      "end": 7707
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 7728,
                                        "end": 7739
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 7728,
                                      "end": 7739
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 5769,
                                  "end": 7757
                                }
                              },
                              "loc": {
                                "start": 5759,
                                "end": 7757
                              }
                            }
                          ],
                          "loc": {
                            "start": 5279,
                            "end": 7771
                          }
                        },
                        "loc": {
                          "start": 5271,
                          "end": 7771
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "labels",
                          "loc": {
                            "start": 7784,
                            "end": 7790
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
                                  "start": 7809,
                                  "end": 7811
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7809,
                                "end": 7811
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "color",
                                "loc": {
                                  "start": 7828,
                                  "end": 7833
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7828,
                                "end": 7833
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "label",
                                "loc": {
                                  "start": 7850,
                                  "end": 7855
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7850,
                                "end": 7855
                              }
                            }
                          ],
                          "loc": {
                            "start": 7791,
                            "end": 7869
                          }
                        },
                        "loc": {
                          "start": 7784,
                          "end": 7869
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "reminderList",
                          "loc": {
                            "start": 7882,
                            "end": 7894
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
                                  "start": 7913,
                                  "end": 7915
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7913,
                                "end": 7915
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 7932,
                                  "end": 7942
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7932,
                                "end": 7942
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "updated_at",
                                "loc": {
                                  "start": 7959,
                                  "end": 7969
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7959,
                                "end": 7969
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "reminders",
                                "loc": {
                                  "start": 7986,
                                  "end": 7995
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
                                        "start": 8018,
                                        "end": 8020
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8018,
                                      "end": 8020
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 8041,
                                        "end": 8051
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8041,
                                      "end": 8051
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 8072,
                                        "end": 8082
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8072,
                                      "end": 8082
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 8103,
                                        "end": 8107
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8103,
                                      "end": 8107
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 8128,
                                        "end": 8139
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8128,
                                      "end": 8139
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "dueDate",
                                      "loc": {
                                        "start": 8160,
                                        "end": 8167
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8160,
                                      "end": 8167
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "index",
                                      "loc": {
                                        "start": 8188,
                                        "end": 8193
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8188,
                                      "end": 8193
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isComplete",
                                      "loc": {
                                        "start": 8214,
                                        "end": 8224
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8214,
                                      "end": 8224
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "reminderItems",
                                      "loc": {
                                        "start": 8245,
                                        "end": 8258
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
                                              "start": 8285,
                                              "end": 8287
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 8285,
                                            "end": 8287
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 8312,
                                              "end": 8322
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 8312,
                                            "end": 8322
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "updated_at",
                                            "loc": {
                                              "start": 8347,
                                              "end": 8357
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 8347,
                                            "end": 8357
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 8382,
                                              "end": 8386
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 8382,
                                            "end": 8386
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 8411,
                                              "end": 8422
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 8411,
                                            "end": 8422
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "dueDate",
                                            "loc": {
                                              "start": 8447,
                                              "end": 8454
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 8447,
                                            "end": 8454
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "index",
                                            "loc": {
                                              "start": 8479,
                                              "end": 8484
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 8479,
                                            "end": 8484
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isComplete",
                                            "loc": {
                                              "start": 8509,
                                              "end": 8519
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 8509,
                                            "end": 8519
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 8259,
                                        "end": 8541
                                      }
                                    },
                                    "loc": {
                                      "start": 8245,
                                      "end": 8541
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 7996,
                                  "end": 8559
                                }
                              },
                              "loc": {
                                "start": 7986,
                                "end": 8559
                              }
                            }
                          ],
                          "loc": {
                            "start": 7895,
                            "end": 8573
                          }
                        },
                        "loc": {
                          "start": 7882,
                          "end": 8573
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "resourceList",
                          "loc": {
                            "start": 8586,
                            "end": 8598
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
                                  "start": 8617,
                                  "end": 8619
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 8617,
                                "end": 8619
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 8636,
                                  "end": 8646
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 8636,
                                "end": 8646
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "translations",
                                "loc": {
                                  "start": 8663,
                                  "end": 8675
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
                                        "start": 8698,
                                        "end": 8700
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8698,
                                      "end": 8700
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "language",
                                      "loc": {
                                        "start": 8721,
                                        "end": 8729
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8721,
                                      "end": 8729
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 8750,
                                        "end": 8761
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8750,
                                      "end": 8761
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 8782,
                                        "end": 8786
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8782,
                                      "end": 8786
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 8676,
                                  "end": 8804
                                }
                              },
                              "loc": {
                                "start": 8663,
                                "end": 8804
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "resources",
                                "loc": {
                                  "start": 8821,
                                  "end": 8830
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
                                        "start": 8853,
                                        "end": 8855
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8853,
                                      "end": 8855
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "index",
                                      "loc": {
                                        "start": 8876,
                                        "end": 8881
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8876,
                                      "end": 8881
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "link",
                                      "loc": {
                                        "start": 8902,
                                        "end": 8906
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8902,
                                      "end": 8906
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "usedFor",
                                      "loc": {
                                        "start": 8927,
                                        "end": 8934
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8927,
                                      "end": 8934
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "translations",
                                      "loc": {
                                        "start": 8955,
                                        "end": 8967
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
                                              "start": 8994,
                                              "end": 8996
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 8994,
                                            "end": 8996
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "language",
                                            "loc": {
                                              "start": 9021,
                                              "end": 9029
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 9021,
                                            "end": 9029
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 9054,
                                              "end": 9065
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 9054,
                                            "end": 9065
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 9090,
                                              "end": 9094
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 9090,
                                            "end": 9094
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 8968,
                                        "end": 9116
                                      }
                                    },
                                    "loc": {
                                      "start": 8955,
                                      "end": 9116
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 8831,
                                  "end": 9134
                                }
                              },
                              "loc": {
                                "start": 8821,
                                "end": 9134
                              }
                            }
                          ],
                          "loc": {
                            "start": 8599,
                            "end": 9148
                          }
                        },
                        "loc": {
                          "start": 8586,
                          "end": 9148
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "schedule",
                          "loc": {
                            "start": 9161,
                            "end": 9169
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
                                  "start": 9191,
                                  "end": 9206
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 9188,
                                "end": 9206
                              }
                            }
                          ],
                          "loc": {
                            "start": 9170,
                            "end": 9220
                          }
                        },
                        "loc": {
                          "start": 9161,
                          "end": 9220
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 9233,
                            "end": 9235
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 9233,
                          "end": 9235
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 9248,
                            "end": 9252
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 9248,
                          "end": 9252
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 9265,
                            "end": 9276
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 9265,
                          "end": 9276
                        }
                      }
                    ],
                    "loc": {
                      "start": 5257,
                      "end": 9286
                    }
                  },
                  "loc": {
                    "start": 5246,
                    "end": 9286
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "handle",
                    "loc": {
                      "start": 9295,
                      "end": 9301
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9295,
                    "end": 9301
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "hasPremium",
                    "loc": {
                      "start": 9310,
                      "end": 9320
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9310,
                    "end": 9320
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 9329,
                      "end": 9331
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9329,
                    "end": 9331
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "languages",
                    "loc": {
                      "start": 9340,
                      "end": 9349
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9340,
                    "end": 9349
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "membershipsCount",
                    "loc": {
                      "start": 9358,
                      "end": 9374
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9358,
                    "end": 9374
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 9383,
                      "end": 9387
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9383,
                    "end": 9387
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "notesCount",
                    "loc": {
                      "start": 9396,
                      "end": 9406
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9396,
                    "end": 9406
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "projectsCount",
                    "loc": {
                      "start": 9415,
                      "end": 9428
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9415,
                    "end": 9428
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "questionsAskedCount",
                    "loc": {
                      "start": 9437,
                      "end": 9456
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9437,
                    "end": 9456
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "routinesCount",
                    "loc": {
                      "start": 9465,
                      "end": 9478
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9465,
                    "end": 9478
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "smartContractsCount",
                    "loc": {
                      "start": 9487,
                      "end": 9506
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9487,
                    "end": 9506
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "standardsCount",
                    "loc": {
                      "start": 9515,
                      "end": 9529
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9515,
                    "end": 9529
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "theme",
                    "loc": {
                      "start": 9538,
                      "end": 9543
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9538,
                    "end": 9543
                  }
                }
              ],
              "loc": {
                "start": 396,
                "end": 9549
              }
            },
            "loc": {
              "start": 390,
              "end": 9549
            }
          }
        ],
        "loc": {
          "start": 356,
          "end": 9553
        }
      },
      "loc": {
        "start": 329,
        "end": 9553
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 40,
          "end": 42
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 40,
        "end": 42
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 43,
          "end": 53
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 43,
        "end": 53
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 54,
          "end": 64
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 54,
        "end": 64
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "startTime",
        "loc": {
          "start": 65,
          "end": 74
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 65,
        "end": 74
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "endTime",
        "loc": {
          "start": 75,
          "end": 82
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 75,
        "end": 82
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "timezone",
        "loc": {
          "start": 83,
          "end": 91
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 83,
        "end": 91
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "exceptions",
        "loc": {
          "start": 92,
          "end": 102
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
                "start": 109,
                "end": 111
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 109,
              "end": 111
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "originalStartTime",
              "loc": {
                "start": 116,
                "end": 133
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 116,
              "end": 133
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "newStartTime",
              "loc": {
                "start": 138,
                "end": 150
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 138,
              "end": 150
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "newEndTime",
              "loc": {
                "start": 155,
                "end": 165
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 155,
              "end": 165
            }
          }
        ],
        "loc": {
          "start": 103,
          "end": 167
        }
      },
      "loc": {
        "start": 92,
        "end": 167
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "recurrences",
        "loc": {
          "start": 168,
          "end": 179
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
                "start": 186,
                "end": 188
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 186,
              "end": 188
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "recurrenceType",
              "loc": {
                "start": 193,
                "end": 207
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 193,
              "end": 207
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "interval",
              "loc": {
                "start": 212,
                "end": 220
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 212,
              "end": 220
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dayOfWeek",
              "loc": {
                "start": 225,
                "end": 234
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 225,
              "end": 234
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dayOfMonth",
              "loc": {
                "start": 239,
                "end": 249
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 239,
              "end": 249
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "month",
              "loc": {
                "start": 254,
                "end": 259
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 254,
              "end": 259
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "endDate",
              "loc": {
                "start": 264,
                "end": 271
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 264,
              "end": 271
            }
          }
        ],
        "loc": {
          "start": 180,
          "end": 273
        }
      },
      "loc": {
        "start": 168,
        "end": 273
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
          "start": 10,
          "end": 25
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Schedule",
          "loc": {
            "start": 29,
            "end": 37
          }
        },
        "loc": {
          "start": 29,
          "end": 37
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
                "start": 40,
                "end": 42
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 40,
              "end": 42
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 43,
                "end": 53
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 43,
              "end": 53
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 54,
                "end": 64
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 54,
              "end": 64
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "startTime",
              "loc": {
                "start": 65,
                "end": 74
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 65,
              "end": 74
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "endTime",
              "loc": {
                "start": 75,
                "end": 82
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 75,
              "end": 82
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timezone",
              "loc": {
                "start": 83,
                "end": 91
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 83,
              "end": 91
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "exceptions",
              "loc": {
                "start": 92,
                "end": 102
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
                      "start": 109,
                      "end": 111
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 109,
                    "end": 111
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "originalStartTime",
                    "loc": {
                      "start": 116,
                      "end": 133
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 116,
                    "end": 133
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newStartTime",
                    "loc": {
                      "start": 138,
                      "end": 150
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 138,
                    "end": 150
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newEndTime",
                    "loc": {
                      "start": 155,
                      "end": 165
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 155,
                    "end": 165
                  }
                }
              ],
              "loc": {
                "start": 103,
                "end": 167
              }
            },
            "loc": {
              "start": 92,
              "end": 167
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "recurrences",
              "loc": {
                "start": 168,
                "end": 179
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
                      "start": 186,
                      "end": 188
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 186,
                    "end": 188
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "recurrenceType",
                    "loc": {
                      "start": 193,
                      "end": 207
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 193,
                    "end": 207
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "interval",
                    "loc": {
                      "start": 212,
                      "end": 220
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 212,
                    "end": 220
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfWeek",
                    "loc": {
                      "start": 225,
                      "end": 234
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 225,
                    "end": 234
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfMonth",
                    "loc": {
                      "start": 239,
                      "end": 249
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 239,
                    "end": 249
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "month",
                    "loc": {
                      "start": 254,
                      "end": 259
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 254,
                    "end": 259
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endDate",
                    "loc": {
                      "start": 264,
                      "end": 271
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 264,
                    "end": 271
                  }
                }
              ],
              "loc": {
                "start": 180,
                "end": 273
              }
            },
            "loc": {
              "start": 168,
              "end": 273
            }
          }
        ],
        "loc": {
          "start": 38,
          "end": 275
        }
      },
      "loc": {
        "start": 1,
        "end": 275
      }
    }
  },
  "rootValue": {},
  "operation": {
    "kind": "OperationDefinition",
    "operation": "mutation",
    "name": {
      "kind": "Name",
      "value": "emailSignUp",
      "loc": {
        "start": 286,
        "end": 297
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
              "start": 299,
              "end": 304
            }
          },
          "loc": {
            "start": 298,
            "end": 304
          }
        },
        "type": {
          "kind": "NonNullType",
          "type": {
            "kind": "NamedType",
            "name": {
              "kind": "Name",
              "value": "EmailSignUpInput",
              "loc": {
                "start": 306,
                "end": 322
              }
            },
            "loc": {
              "start": 306,
              "end": 322
            }
          },
          "loc": {
            "start": 306,
            "end": 323
          }
        },
        "directives": [],
        "loc": {
          "start": 298,
          "end": 323
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
            "value": "emailSignUp",
            "loc": {
              "start": 329,
              "end": 340
            }
          },
          "arguments": [
            {
              "kind": "Argument",
              "name": {
                "kind": "Name",
                "value": "input",
                "loc": {
                  "start": 341,
                  "end": 346
                }
              },
              "value": {
                "kind": "Variable",
                "name": {
                  "kind": "Name",
                  "value": "input",
                  "loc": {
                    "start": 349,
                    "end": 354
                  }
                },
                "loc": {
                  "start": 348,
                  "end": 354
                }
              },
              "loc": {
                "start": 341,
                "end": 354
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
                    "start": 362,
                    "end": 372
                  }
                },
                "arguments": [],
                "directives": [],
                "loc": {
                  "start": 362,
                  "end": 372
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "timeZone",
                  "loc": {
                    "start": 377,
                    "end": 385
                  }
                },
                "arguments": [],
                "directives": [],
                "loc": {
                  "start": 377,
                  "end": 385
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "users",
                  "loc": {
                    "start": 390,
                    "end": 395
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
                          "start": 406,
                          "end": 421
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
                                "start": 436,
                                "end": 440
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
                                      "start": 459,
                                      "end": 466
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
                                            "start": 489,
                                            "end": 491
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 489,
                                          "end": 491
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "filterType",
                                          "loc": {
                                            "start": 512,
                                            "end": 522
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 512,
                                          "end": 522
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "tag",
                                          "loc": {
                                            "start": 543,
                                            "end": 546
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
                                                  "start": 573,
                                                  "end": 575
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 573,
                                                "end": 575
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "created_at",
                                                "loc": {
                                                  "start": 600,
                                                  "end": 610
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 600,
                                                "end": 610
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "tag",
                                                "loc": {
                                                  "start": 635,
                                                  "end": 638
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 635,
                                                "end": 638
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "bookmarks",
                                                "loc": {
                                                  "start": 663,
                                                  "end": 672
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 663,
                                                "end": 672
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "translations",
                                                "loc": {
                                                  "start": 697,
                                                  "end": 709
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
                                                        "start": 740,
                                                        "end": 742
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 740,
                                                      "end": 742
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "language",
                                                      "loc": {
                                                        "start": 771,
                                                        "end": 779
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 771,
                                                      "end": 779
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "description",
                                                      "loc": {
                                                        "start": 808,
                                                        "end": 819
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 808,
                                                      "end": 819
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 710,
                                                  "end": 845
                                                }
                                              },
                                              "loc": {
                                                "start": 697,
                                                "end": 845
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "you",
                                                "loc": {
                                                  "start": 870,
                                                  "end": 873
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
                                                        "start": 904,
                                                        "end": 909
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 904,
                                                      "end": 909
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "isBookmarked",
                                                      "loc": {
                                                        "start": 938,
                                                        "end": 950
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 938,
                                                      "end": 950
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 874,
                                                  "end": 976
                                                }
                                              },
                                              "loc": {
                                                "start": 870,
                                                "end": 976
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 547,
                                            "end": 998
                                          }
                                        },
                                        "loc": {
                                          "start": 543,
                                          "end": 998
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "focusMode",
                                          "loc": {
                                            "start": 1019,
                                            "end": 1028
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
                                                  "start": 1055,
                                                  "end": 1061
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
                                                        "start": 1092,
                                                        "end": 1094
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 1092,
                                                      "end": 1094
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "color",
                                                      "loc": {
                                                        "start": 1123,
                                                        "end": 1128
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 1123,
                                                      "end": 1128
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "label",
                                                      "loc": {
                                                        "start": 1157,
                                                        "end": 1162
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 1157,
                                                      "end": 1162
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 1062,
                                                  "end": 1188
                                                }
                                              },
                                              "loc": {
                                                "start": 1055,
                                                "end": 1188
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "reminderList",
                                                "loc": {
                                                  "start": 1213,
                                                  "end": 1225
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
                                                        "start": 1256,
                                                        "end": 1258
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 1256,
                                                      "end": 1258
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "created_at",
                                                      "loc": {
                                                        "start": 1287,
                                                        "end": 1297
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 1287,
                                                      "end": 1297
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "updated_at",
                                                      "loc": {
                                                        "start": 1326,
                                                        "end": 1336
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 1326,
                                                      "end": 1336
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "reminders",
                                                      "loc": {
                                                        "start": 1365,
                                                        "end": 1374
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
                                                              "start": 1409,
                                                              "end": 1411
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 1409,
                                                            "end": 1411
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "created_at",
                                                            "loc": {
                                                              "start": 1444,
                                                              "end": 1454
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 1444,
                                                            "end": 1454
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "updated_at",
                                                            "loc": {
                                                              "start": 1487,
                                                              "end": 1497
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 1487,
                                                            "end": 1497
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "name",
                                                            "loc": {
                                                              "start": 1530,
                                                              "end": 1534
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 1530,
                                                            "end": 1534
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "description",
                                                            "loc": {
                                                              "start": 1567,
                                                              "end": 1578
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 1567,
                                                            "end": 1578
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "dueDate",
                                                            "loc": {
                                                              "start": 1611,
                                                              "end": 1618
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 1611,
                                                            "end": 1618
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "index",
                                                            "loc": {
                                                              "start": 1651,
                                                              "end": 1656
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 1651,
                                                            "end": 1656
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "isComplete",
                                                            "loc": {
                                                              "start": 1689,
                                                              "end": 1699
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 1689,
                                                            "end": 1699
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "reminderItems",
                                                            "loc": {
                                                              "start": 1732,
                                                              "end": 1745
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
                                                                    "start": 1784,
                                                                    "end": 1786
                                                                  }
                                                                },
                                                                "arguments": [],
                                                                "directives": [],
                                                                "loc": {
                                                                  "start": 1784,
                                                                  "end": 1786
                                                                }
                                                              },
                                                              {
                                                                "kind": "Field",
                                                                "name": {
                                                                  "kind": "Name",
                                                                  "value": "created_at",
                                                                  "loc": {
                                                                    "start": 1823,
                                                                    "end": 1833
                                                                  }
                                                                },
                                                                "arguments": [],
                                                                "directives": [],
                                                                "loc": {
                                                                  "start": 1823,
                                                                  "end": 1833
                                                                }
                                                              },
                                                              {
                                                                "kind": "Field",
                                                                "name": {
                                                                  "kind": "Name",
                                                                  "value": "updated_at",
                                                                  "loc": {
                                                                    "start": 1870,
                                                                    "end": 1880
                                                                  }
                                                                },
                                                                "arguments": [],
                                                                "directives": [],
                                                                "loc": {
                                                                  "start": 1870,
                                                                  "end": 1880
                                                                }
                                                              },
                                                              {
                                                                "kind": "Field",
                                                                "name": {
                                                                  "kind": "Name",
                                                                  "value": "name",
                                                                  "loc": {
                                                                    "start": 1917,
                                                                    "end": 1921
                                                                  }
                                                                },
                                                                "arguments": [],
                                                                "directives": [],
                                                                "loc": {
                                                                  "start": 1917,
                                                                  "end": 1921
                                                                }
                                                              },
                                                              {
                                                                "kind": "Field",
                                                                "name": {
                                                                  "kind": "Name",
                                                                  "value": "description",
                                                                  "loc": {
                                                                    "start": 1958,
                                                                    "end": 1969
                                                                  }
                                                                },
                                                                "arguments": [],
                                                                "directives": [],
                                                                "loc": {
                                                                  "start": 1958,
                                                                  "end": 1969
                                                                }
                                                              },
                                                              {
                                                                "kind": "Field",
                                                                "name": {
                                                                  "kind": "Name",
                                                                  "value": "dueDate",
                                                                  "loc": {
                                                                    "start": 2006,
                                                                    "end": 2013
                                                                  }
                                                                },
                                                                "arguments": [],
                                                                "directives": [],
                                                                "loc": {
                                                                  "start": 2006,
                                                                  "end": 2013
                                                                }
                                                              },
                                                              {
                                                                "kind": "Field",
                                                                "name": {
                                                                  "kind": "Name",
                                                                  "value": "index",
                                                                  "loc": {
                                                                    "start": 2050,
                                                                    "end": 2055
                                                                  }
                                                                },
                                                                "arguments": [],
                                                                "directives": [],
                                                                "loc": {
                                                                  "start": 2050,
                                                                  "end": 2055
                                                                }
                                                              },
                                                              {
                                                                "kind": "Field",
                                                                "name": {
                                                                  "kind": "Name",
                                                                  "value": "isComplete",
                                                                  "loc": {
                                                                    "start": 2092,
                                                                    "end": 2102
                                                                  }
                                                                },
                                                                "arguments": [],
                                                                "directives": [],
                                                                "loc": {
                                                                  "start": 2092,
                                                                  "end": 2102
                                                                }
                                                              }
                                                            ],
                                                            "loc": {
                                                              "start": 1746,
                                                              "end": 2136
                                                            }
                                                          },
                                                          "loc": {
                                                            "start": 1732,
                                                            "end": 2136
                                                          }
                                                        }
                                                      ],
                                                      "loc": {
                                                        "start": 1375,
                                                        "end": 2166
                                                      }
                                                    },
                                                    "loc": {
                                                      "start": 1365,
                                                      "end": 2166
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 1226,
                                                  "end": 2192
                                                }
                                              },
                                              "loc": {
                                                "start": 1213,
                                                "end": 2192
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "resourceList",
                                                "loc": {
                                                  "start": 2217,
                                                  "end": 2229
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
                                                        "start": 2260,
                                                        "end": 2262
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 2260,
                                                      "end": 2262
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "created_at",
                                                      "loc": {
                                                        "start": 2291,
                                                        "end": 2301
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 2291,
                                                      "end": 2301
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "translations",
                                                      "loc": {
                                                        "start": 2330,
                                                        "end": 2342
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
                                                              "start": 2377,
                                                              "end": 2379
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 2377,
                                                            "end": 2379
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "language",
                                                            "loc": {
                                                              "start": 2412,
                                                              "end": 2420
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 2412,
                                                            "end": 2420
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "description",
                                                            "loc": {
                                                              "start": 2453,
                                                              "end": 2464
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 2453,
                                                            "end": 2464
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "name",
                                                            "loc": {
                                                              "start": 2497,
                                                              "end": 2501
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 2497,
                                                            "end": 2501
                                                          }
                                                        }
                                                      ],
                                                      "loc": {
                                                        "start": 2343,
                                                        "end": 2531
                                                      }
                                                    },
                                                    "loc": {
                                                      "start": 2330,
                                                      "end": 2531
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "resources",
                                                      "loc": {
                                                        "start": 2560,
                                                        "end": 2569
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
                                                              "start": 2604,
                                                              "end": 2606
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 2604,
                                                            "end": 2606
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "index",
                                                            "loc": {
                                                              "start": 2639,
                                                              "end": 2644
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 2639,
                                                            "end": 2644
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "link",
                                                            "loc": {
                                                              "start": 2677,
                                                              "end": 2681
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 2677,
                                                            "end": 2681
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "usedFor",
                                                            "loc": {
                                                              "start": 2714,
                                                              "end": 2721
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 2714,
                                                            "end": 2721
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "translations",
                                                            "loc": {
                                                              "start": 2754,
                                                              "end": 2766
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
                                                                    "start": 2805,
                                                                    "end": 2807
                                                                  }
                                                                },
                                                                "arguments": [],
                                                                "directives": [],
                                                                "loc": {
                                                                  "start": 2805,
                                                                  "end": 2807
                                                                }
                                                              },
                                                              {
                                                                "kind": "Field",
                                                                "name": {
                                                                  "kind": "Name",
                                                                  "value": "language",
                                                                  "loc": {
                                                                    "start": 2844,
                                                                    "end": 2852
                                                                  }
                                                                },
                                                                "arguments": [],
                                                                "directives": [],
                                                                "loc": {
                                                                  "start": 2844,
                                                                  "end": 2852
                                                                }
                                                              },
                                                              {
                                                                "kind": "Field",
                                                                "name": {
                                                                  "kind": "Name",
                                                                  "value": "description",
                                                                  "loc": {
                                                                    "start": 2889,
                                                                    "end": 2900
                                                                  }
                                                                },
                                                                "arguments": [],
                                                                "directives": [],
                                                                "loc": {
                                                                  "start": 2889,
                                                                  "end": 2900
                                                                }
                                                              },
                                                              {
                                                                "kind": "Field",
                                                                "name": {
                                                                  "kind": "Name",
                                                                  "value": "name",
                                                                  "loc": {
                                                                    "start": 2937,
                                                                    "end": 2941
                                                                  }
                                                                },
                                                                "arguments": [],
                                                                "directives": [],
                                                                "loc": {
                                                                  "start": 2937,
                                                                  "end": 2941
                                                                }
                                                              }
                                                            ],
                                                            "loc": {
                                                              "start": 2767,
                                                              "end": 2975
                                                            }
                                                          },
                                                          "loc": {
                                                            "start": 2754,
                                                            "end": 2975
                                                          }
                                                        }
                                                      ],
                                                      "loc": {
                                                        "start": 2570,
                                                        "end": 3005
                                                      }
                                                    },
                                                    "loc": {
                                                      "start": 2560,
                                                      "end": 3005
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 2230,
                                                  "end": 3031
                                                }
                                              },
                                              "loc": {
                                                "start": 2217,
                                                "end": 3031
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "schedule",
                                                "loc": {
                                                  "start": 3056,
                                                  "end": 3064
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
                                                        "start": 3098,
                                                        "end": 3113
                                                      }
                                                    },
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 3095,
                                                      "end": 3113
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 3065,
                                                  "end": 3139
                                                }
                                              },
                                              "loc": {
                                                "start": 3056,
                                                "end": 3139
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "id",
                                                "loc": {
                                                  "start": 3164,
                                                  "end": 3166
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3164,
                                                "end": 3166
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "name",
                                                "loc": {
                                                  "start": 3191,
                                                  "end": 3195
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3191,
                                                "end": 3195
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "description",
                                                "loc": {
                                                  "start": 3220,
                                                  "end": 3231
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3220,
                                                "end": 3231
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 1029,
                                            "end": 3253
                                          }
                                        },
                                        "loc": {
                                          "start": 1019,
                                          "end": 3253
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 467,
                                      "end": 3271
                                    }
                                  },
                                  "loc": {
                                    "start": 459,
                                    "end": 3271
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "labels",
                                    "loc": {
                                      "start": 3288,
                                      "end": 3294
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
                                            "start": 3317,
                                            "end": 3319
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 3317,
                                          "end": 3319
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "color",
                                          "loc": {
                                            "start": 3340,
                                            "end": 3345
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 3340,
                                          "end": 3345
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "label",
                                          "loc": {
                                            "start": 3366,
                                            "end": 3371
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 3366,
                                          "end": 3371
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 3295,
                                      "end": 3389
                                    }
                                  },
                                  "loc": {
                                    "start": 3288,
                                    "end": 3389
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "reminderList",
                                    "loc": {
                                      "start": 3406,
                                      "end": 3418
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
                                            "start": 3441,
                                            "end": 3443
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 3441,
                                          "end": 3443
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "created_at",
                                          "loc": {
                                            "start": 3464,
                                            "end": 3474
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 3464,
                                          "end": 3474
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "updated_at",
                                          "loc": {
                                            "start": 3495,
                                            "end": 3505
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 3495,
                                          "end": 3505
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "reminders",
                                          "loc": {
                                            "start": 3526,
                                            "end": 3535
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
                                                  "start": 3562,
                                                  "end": 3564
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3562,
                                                "end": 3564
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "created_at",
                                                "loc": {
                                                  "start": 3589,
                                                  "end": 3599
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3589,
                                                "end": 3599
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "updated_at",
                                                "loc": {
                                                  "start": 3624,
                                                  "end": 3634
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3624,
                                                "end": 3634
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "name",
                                                "loc": {
                                                  "start": 3659,
                                                  "end": 3663
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3659,
                                                "end": 3663
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "description",
                                                "loc": {
                                                  "start": 3688,
                                                  "end": 3699
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3688,
                                                "end": 3699
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "dueDate",
                                                "loc": {
                                                  "start": 3724,
                                                  "end": 3731
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3724,
                                                "end": 3731
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "index",
                                                "loc": {
                                                  "start": 3756,
                                                  "end": 3761
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3756,
                                                "end": 3761
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "isComplete",
                                                "loc": {
                                                  "start": 3786,
                                                  "end": 3796
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3786,
                                                "end": 3796
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "reminderItems",
                                                "loc": {
                                                  "start": 3821,
                                                  "end": 3834
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
                                                        "start": 3865,
                                                        "end": 3867
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 3865,
                                                      "end": 3867
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "created_at",
                                                      "loc": {
                                                        "start": 3896,
                                                        "end": 3906
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 3896,
                                                      "end": 3906
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "updated_at",
                                                      "loc": {
                                                        "start": 3935,
                                                        "end": 3945
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 3935,
                                                      "end": 3945
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "name",
                                                      "loc": {
                                                        "start": 3974,
                                                        "end": 3978
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 3974,
                                                      "end": 3978
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "description",
                                                      "loc": {
                                                        "start": 4007,
                                                        "end": 4018
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 4007,
                                                      "end": 4018
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "dueDate",
                                                      "loc": {
                                                        "start": 4047,
                                                        "end": 4054
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 4047,
                                                      "end": 4054
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "index",
                                                      "loc": {
                                                        "start": 4083,
                                                        "end": 4088
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 4083,
                                                      "end": 4088
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "isComplete",
                                                      "loc": {
                                                        "start": 4117,
                                                        "end": 4127
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 4117,
                                                      "end": 4127
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 3835,
                                                  "end": 4153
                                                }
                                              },
                                              "loc": {
                                                "start": 3821,
                                                "end": 4153
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 3536,
                                            "end": 4175
                                          }
                                        },
                                        "loc": {
                                          "start": 3526,
                                          "end": 4175
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 3419,
                                      "end": 4193
                                    }
                                  },
                                  "loc": {
                                    "start": 3406,
                                    "end": 4193
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "resourceList",
                                    "loc": {
                                      "start": 4210,
                                      "end": 4222
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
                                            "start": 4245,
                                            "end": 4247
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 4245,
                                          "end": 4247
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "created_at",
                                          "loc": {
                                            "start": 4268,
                                            "end": 4278
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 4268,
                                          "end": 4278
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "translations",
                                          "loc": {
                                            "start": 4299,
                                            "end": 4311
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
                                                  "start": 4338,
                                                  "end": 4340
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4338,
                                                "end": 4340
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "language",
                                                "loc": {
                                                  "start": 4365,
                                                  "end": 4373
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4365,
                                                "end": 4373
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "description",
                                                "loc": {
                                                  "start": 4398,
                                                  "end": 4409
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4398,
                                                "end": 4409
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "name",
                                                "loc": {
                                                  "start": 4434,
                                                  "end": 4438
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4434,
                                                "end": 4438
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 4312,
                                            "end": 4460
                                          }
                                        },
                                        "loc": {
                                          "start": 4299,
                                          "end": 4460
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "resources",
                                          "loc": {
                                            "start": 4481,
                                            "end": 4490
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
                                                  "start": 4517,
                                                  "end": 4519
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4517,
                                                "end": 4519
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "index",
                                                "loc": {
                                                  "start": 4544,
                                                  "end": 4549
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4544,
                                                "end": 4549
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "link",
                                                "loc": {
                                                  "start": 4574,
                                                  "end": 4578
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4574,
                                                "end": 4578
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "usedFor",
                                                "loc": {
                                                  "start": 4603,
                                                  "end": 4610
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4603,
                                                "end": 4610
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "translations",
                                                "loc": {
                                                  "start": 4635,
                                                  "end": 4647
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
                                                        "start": 4678,
                                                        "end": 4680
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 4678,
                                                      "end": 4680
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "language",
                                                      "loc": {
                                                        "start": 4709,
                                                        "end": 4717
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 4709,
                                                      "end": 4717
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "description",
                                                      "loc": {
                                                        "start": 4746,
                                                        "end": 4757
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 4746,
                                                      "end": 4757
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "name",
                                                      "loc": {
                                                        "start": 4786,
                                                        "end": 4790
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 4786,
                                                      "end": 4790
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 4648,
                                                  "end": 4816
                                                }
                                              },
                                              "loc": {
                                                "start": 4635,
                                                "end": 4816
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 4491,
                                            "end": 4838
                                          }
                                        },
                                        "loc": {
                                          "start": 4481,
                                          "end": 4838
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 4223,
                                      "end": 4856
                                    }
                                  },
                                  "loc": {
                                    "start": 4210,
                                    "end": 4856
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "schedule",
                                    "loc": {
                                      "start": 4873,
                                      "end": 4881
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
                                            "start": 4907,
                                            "end": 4922
                                          }
                                        },
                                        "directives": [],
                                        "loc": {
                                          "start": 4904,
                                          "end": 4922
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 4882,
                                      "end": 4940
                                    }
                                  },
                                  "loc": {
                                    "start": 4873,
                                    "end": 4940
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "id",
                                    "loc": {
                                      "start": 4957,
                                      "end": 4959
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 4957,
                                    "end": 4959
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "name",
                                    "loc": {
                                      "start": 4976,
                                      "end": 4980
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 4976,
                                    "end": 4980
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "description",
                                    "loc": {
                                      "start": 4997,
                                      "end": 5008
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 4997,
                                    "end": 5008
                                  }
                                }
                              ],
                              "loc": {
                                "start": 441,
                                "end": 5022
                              }
                            },
                            "loc": {
                              "start": 436,
                              "end": 5022
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "stopCondition",
                              "loc": {
                                "start": 5035,
                                "end": 5048
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 5035,
                              "end": 5048
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "stopTime",
                              "loc": {
                                "start": 5061,
                                "end": 5069
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 5061,
                              "end": 5069
                            }
                          }
                        ],
                        "loc": {
                          "start": 422,
                          "end": 5079
                        }
                      },
                      "loc": {
                        "start": 406,
                        "end": 5079
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "apisCount",
                        "loc": {
                          "start": 5088,
                          "end": 5097
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 5088,
                        "end": 5097
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "bookmarkLists",
                        "loc": {
                          "start": 5106,
                          "end": 5119
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
                                "start": 5134,
                                "end": 5136
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 5134,
                              "end": 5136
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "created_at",
                              "loc": {
                                "start": 5149,
                                "end": 5159
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 5149,
                              "end": 5159
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "updated_at",
                              "loc": {
                                "start": 5172,
                                "end": 5182
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 5172,
                              "end": 5182
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "label",
                              "loc": {
                                "start": 5195,
                                "end": 5200
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 5195,
                              "end": 5200
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "bookmarksCount",
                              "loc": {
                                "start": 5213,
                                "end": 5227
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 5213,
                              "end": 5227
                            }
                          }
                        ],
                        "loc": {
                          "start": 5120,
                          "end": 5237
                        }
                      },
                      "loc": {
                        "start": 5106,
                        "end": 5237
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "focusModes",
                        "loc": {
                          "start": 5246,
                          "end": 5256
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
                                "start": 5271,
                                "end": 5278
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
                                      "start": 5297,
                                      "end": 5299
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 5297,
                                    "end": 5299
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "filterType",
                                    "loc": {
                                      "start": 5316,
                                      "end": 5326
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 5316,
                                    "end": 5326
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "tag",
                                    "loc": {
                                      "start": 5343,
                                      "end": 5346
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
                                            "start": 5369,
                                            "end": 5371
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 5369,
                                          "end": 5371
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "created_at",
                                          "loc": {
                                            "start": 5392,
                                            "end": 5402
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 5392,
                                          "end": 5402
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "tag",
                                          "loc": {
                                            "start": 5423,
                                            "end": 5426
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 5423,
                                          "end": 5426
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "bookmarks",
                                          "loc": {
                                            "start": 5447,
                                            "end": 5456
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 5447,
                                          "end": 5456
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "translations",
                                          "loc": {
                                            "start": 5477,
                                            "end": 5489
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
                                                  "start": 5516,
                                                  "end": 5518
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 5516,
                                                "end": 5518
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "language",
                                                "loc": {
                                                  "start": 5543,
                                                  "end": 5551
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 5543,
                                                "end": 5551
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "description",
                                                "loc": {
                                                  "start": 5576,
                                                  "end": 5587
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 5576,
                                                "end": 5587
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 5490,
                                            "end": 5609
                                          }
                                        },
                                        "loc": {
                                          "start": 5477,
                                          "end": 5609
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "you",
                                          "loc": {
                                            "start": 5630,
                                            "end": 5633
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
                                                  "start": 5660,
                                                  "end": 5665
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 5660,
                                                "end": 5665
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "isBookmarked",
                                                "loc": {
                                                  "start": 5690,
                                                  "end": 5702
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 5690,
                                                "end": 5702
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 5634,
                                            "end": 5724
                                          }
                                        },
                                        "loc": {
                                          "start": 5630,
                                          "end": 5724
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 5347,
                                      "end": 5742
                                    }
                                  },
                                  "loc": {
                                    "start": 5343,
                                    "end": 5742
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "focusMode",
                                    "loc": {
                                      "start": 5759,
                                      "end": 5768
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
                                            "start": 5791,
                                            "end": 5797
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
                                                  "start": 5824,
                                                  "end": 5826
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 5824,
                                                "end": 5826
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "color",
                                                "loc": {
                                                  "start": 5851,
                                                  "end": 5856
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 5851,
                                                "end": 5856
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "label",
                                                "loc": {
                                                  "start": 5881,
                                                  "end": 5886
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 5881,
                                                "end": 5886
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 5798,
                                            "end": 5908
                                          }
                                        },
                                        "loc": {
                                          "start": 5791,
                                          "end": 5908
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "reminderList",
                                          "loc": {
                                            "start": 5929,
                                            "end": 5941
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
                                                  "start": 5968,
                                                  "end": 5970
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 5968,
                                                "end": 5970
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "created_at",
                                                "loc": {
                                                  "start": 5995,
                                                  "end": 6005
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 5995,
                                                "end": 6005
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "updated_at",
                                                "loc": {
                                                  "start": 6030,
                                                  "end": 6040
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 6030,
                                                "end": 6040
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "reminders",
                                                "loc": {
                                                  "start": 6065,
                                                  "end": 6074
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
                                                        "start": 6105,
                                                        "end": 6107
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 6105,
                                                      "end": 6107
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "created_at",
                                                      "loc": {
                                                        "start": 6136,
                                                        "end": 6146
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 6136,
                                                      "end": 6146
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "updated_at",
                                                      "loc": {
                                                        "start": 6175,
                                                        "end": 6185
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 6175,
                                                      "end": 6185
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "name",
                                                      "loc": {
                                                        "start": 6214,
                                                        "end": 6218
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 6214,
                                                      "end": 6218
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "description",
                                                      "loc": {
                                                        "start": 6247,
                                                        "end": 6258
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 6247,
                                                      "end": 6258
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "dueDate",
                                                      "loc": {
                                                        "start": 6287,
                                                        "end": 6294
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 6287,
                                                      "end": 6294
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "index",
                                                      "loc": {
                                                        "start": 6323,
                                                        "end": 6328
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 6323,
                                                      "end": 6328
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "isComplete",
                                                      "loc": {
                                                        "start": 6357,
                                                        "end": 6367
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 6357,
                                                      "end": 6367
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "reminderItems",
                                                      "loc": {
                                                        "start": 6396,
                                                        "end": 6409
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
                                                              "start": 6444,
                                                              "end": 6446
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 6444,
                                                            "end": 6446
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "created_at",
                                                            "loc": {
                                                              "start": 6479,
                                                              "end": 6489
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 6479,
                                                            "end": 6489
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "updated_at",
                                                            "loc": {
                                                              "start": 6522,
                                                              "end": 6532
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 6522,
                                                            "end": 6532
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "name",
                                                            "loc": {
                                                              "start": 6565,
                                                              "end": 6569
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 6565,
                                                            "end": 6569
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "description",
                                                            "loc": {
                                                              "start": 6602,
                                                              "end": 6613
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 6602,
                                                            "end": 6613
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "dueDate",
                                                            "loc": {
                                                              "start": 6646,
                                                              "end": 6653
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 6646,
                                                            "end": 6653
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "index",
                                                            "loc": {
                                                              "start": 6686,
                                                              "end": 6691
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 6686,
                                                            "end": 6691
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "isComplete",
                                                            "loc": {
                                                              "start": 6724,
                                                              "end": 6734
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 6724,
                                                            "end": 6734
                                                          }
                                                        }
                                                      ],
                                                      "loc": {
                                                        "start": 6410,
                                                        "end": 6764
                                                      }
                                                    },
                                                    "loc": {
                                                      "start": 6396,
                                                      "end": 6764
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 6075,
                                                  "end": 6790
                                                }
                                              },
                                              "loc": {
                                                "start": 6065,
                                                "end": 6790
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 5942,
                                            "end": 6812
                                          }
                                        },
                                        "loc": {
                                          "start": 5929,
                                          "end": 6812
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "resourceList",
                                          "loc": {
                                            "start": 6833,
                                            "end": 6845
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
                                                  "start": 6872,
                                                  "end": 6874
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 6872,
                                                "end": 6874
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "created_at",
                                                "loc": {
                                                  "start": 6899,
                                                  "end": 6909
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 6899,
                                                "end": 6909
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "translations",
                                                "loc": {
                                                  "start": 6934,
                                                  "end": 6946
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
                                                        "start": 6977,
                                                        "end": 6979
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 6977,
                                                      "end": 6979
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "language",
                                                      "loc": {
                                                        "start": 7008,
                                                        "end": 7016
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 7008,
                                                      "end": 7016
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "description",
                                                      "loc": {
                                                        "start": 7045,
                                                        "end": 7056
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 7045,
                                                      "end": 7056
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "name",
                                                      "loc": {
                                                        "start": 7085,
                                                        "end": 7089
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 7085,
                                                      "end": 7089
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 6947,
                                                  "end": 7115
                                                }
                                              },
                                              "loc": {
                                                "start": 6934,
                                                "end": 7115
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "resources",
                                                "loc": {
                                                  "start": 7140,
                                                  "end": 7149
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
                                                        "start": 7180,
                                                        "end": 7182
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 7180,
                                                      "end": 7182
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "index",
                                                      "loc": {
                                                        "start": 7211,
                                                        "end": 7216
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 7211,
                                                      "end": 7216
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "link",
                                                      "loc": {
                                                        "start": 7245,
                                                        "end": 7249
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 7245,
                                                      "end": 7249
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "usedFor",
                                                      "loc": {
                                                        "start": 7278,
                                                        "end": 7285
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 7278,
                                                      "end": 7285
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "translations",
                                                      "loc": {
                                                        "start": 7314,
                                                        "end": 7326
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
                                                              "start": 7361,
                                                              "end": 7363
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 7361,
                                                            "end": 7363
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "language",
                                                            "loc": {
                                                              "start": 7396,
                                                              "end": 7404
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 7396,
                                                            "end": 7404
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "description",
                                                            "loc": {
                                                              "start": 7437,
                                                              "end": 7448
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 7437,
                                                            "end": 7448
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "name",
                                                            "loc": {
                                                              "start": 7481,
                                                              "end": 7485
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 7481,
                                                            "end": 7485
                                                          }
                                                        }
                                                      ],
                                                      "loc": {
                                                        "start": 7327,
                                                        "end": 7515
                                                      }
                                                    },
                                                    "loc": {
                                                      "start": 7314,
                                                      "end": 7515
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 7150,
                                                  "end": 7541
                                                }
                                              },
                                              "loc": {
                                                "start": 7140,
                                                "end": 7541
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 6846,
                                            "end": 7563
                                          }
                                        },
                                        "loc": {
                                          "start": 6833,
                                          "end": 7563
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "schedule",
                                          "loc": {
                                            "start": 7584,
                                            "end": 7592
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
                                                  "start": 7622,
                                                  "end": 7637
                                                }
                                              },
                                              "directives": [],
                                              "loc": {
                                                "start": 7619,
                                                "end": 7637
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 7593,
                                            "end": 7659
                                          }
                                        },
                                        "loc": {
                                          "start": 7584,
                                          "end": 7659
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "id",
                                          "loc": {
                                            "start": 7680,
                                            "end": 7682
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 7680,
                                          "end": 7682
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "name",
                                          "loc": {
                                            "start": 7703,
                                            "end": 7707
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 7703,
                                          "end": 7707
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "description",
                                          "loc": {
                                            "start": 7728,
                                            "end": 7739
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 7728,
                                          "end": 7739
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 5769,
                                      "end": 7757
                                    }
                                  },
                                  "loc": {
                                    "start": 5759,
                                    "end": 7757
                                  }
                                }
                              ],
                              "loc": {
                                "start": 5279,
                                "end": 7771
                              }
                            },
                            "loc": {
                              "start": 5271,
                              "end": 7771
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "labels",
                              "loc": {
                                "start": 7784,
                                "end": 7790
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
                                      "start": 7809,
                                      "end": 7811
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 7809,
                                    "end": 7811
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "color",
                                    "loc": {
                                      "start": 7828,
                                      "end": 7833
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 7828,
                                    "end": 7833
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "label",
                                    "loc": {
                                      "start": 7850,
                                      "end": 7855
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 7850,
                                    "end": 7855
                                  }
                                }
                              ],
                              "loc": {
                                "start": 7791,
                                "end": 7869
                              }
                            },
                            "loc": {
                              "start": 7784,
                              "end": 7869
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "reminderList",
                              "loc": {
                                "start": 7882,
                                "end": 7894
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
                                      "start": 7913,
                                      "end": 7915
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 7913,
                                    "end": 7915
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "created_at",
                                    "loc": {
                                      "start": 7932,
                                      "end": 7942
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 7932,
                                    "end": 7942
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "updated_at",
                                    "loc": {
                                      "start": 7959,
                                      "end": 7969
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 7959,
                                    "end": 7969
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "reminders",
                                    "loc": {
                                      "start": 7986,
                                      "end": 7995
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
                                            "start": 8018,
                                            "end": 8020
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8018,
                                          "end": 8020
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "created_at",
                                          "loc": {
                                            "start": 8041,
                                            "end": 8051
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8041,
                                          "end": 8051
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "updated_at",
                                          "loc": {
                                            "start": 8072,
                                            "end": 8082
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8072,
                                          "end": 8082
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "name",
                                          "loc": {
                                            "start": 8103,
                                            "end": 8107
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8103,
                                          "end": 8107
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "description",
                                          "loc": {
                                            "start": 8128,
                                            "end": 8139
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8128,
                                          "end": 8139
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "dueDate",
                                          "loc": {
                                            "start": 8160,
                                            "end": 8167
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8160,
                                          "end": 8167
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "index",
                                          "loc": {
                                            "start": 8188,
                                            "end": 8193
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8188,
                                          "end": 8193
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "isComplete",
                                          "loc": {
                                            "start": 8214,
                                            "end": 8224
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8214,
                                          "end": 8224
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "reminderItems",
                                          "loc": {
                                            "start": 8245,
                                            "end": 8258
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
                                                  "start": 8285,
                                                  "end": 8287
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 8285,
                                                "end": 8287
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "created_at",
                                                "loc": {
                                                  "start": 8312,
                                                  "end": 8322
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 8312,
                                                "end": 8322
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "updated_at",
                                                "loc": {
                                                  "start": 8347,
                                                  "end": 8357
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 8347,
                                                "end": 8357
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "name",
                                                "loc": {
                                                  "start": 8382,
                                                  "end": 8386
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 8382,
                                                "end": 8386
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "description",
                                                "loc": {
                                                  "start": 8411,
                                                  "end": 8422
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 8411,
                                                "end": 8422
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "dueDate",
                                                "loc": {
                                                  "start": 8447,
                                                  "end": 8454
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 8447,
                                                "end": 8454
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "index",
                                                "loc": {
                                                  "start": 8479,
                                                  "end": 8484
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 8479,
                                                "end": 8484
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "isComplete",
                                                "loc": {
                                                  "start": 8509,
                                                  "end": 8519
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 8509,
                                                "end": 8519
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 8259,
                                            "end": 8541
                                          }
                                        },
                                        "loc": {
                                          "start": 8245,
                                          "end": 8541
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 7996,
                                      "end": 8559
                                    }
                                  },
                                  "loc": {
                                    "start": 7986,
                                    "end": 8559
                                  }
                                }
                              ],
                              "loc": {
                                "start": 7895,
                                "end": 8573
                              }
                            },
                            "loc": {
                              "start": 7882,
                              "end": 8573
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "resourceList",
                              "loc": {
                                "start": 8586,
                                "end": 8598
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
                                      "start": 8617,
                                      "end": 8619
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 8617,
                                    "end": 8619
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "created_at",
                                    "loc": {
                                      "start": 8636,
                                      "end": 8646
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 8636,
                                    "end": 8646
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "translations",
                                    "loc": {
                                      "start": 8663,
                                      "end": 8675
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
                                            "start": 8698,
                                            "end": 8700
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8698,
                                          "end": 8700
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "language",
                                          "loc": {
                                            "start": 8721,
                                            "end": 8729
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8721,
                                          "end": 8729
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "description",
                                          "loc": {
                                            "start": 8750,
                                            "end": 8761
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8750,
                                          "end": 8761
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "name",
                                          "loc": {
                                            "start": 8782,
                                            "end": 8786
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8782,
                                          "end": 8786
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 8676,
                                      "end": 8804
                                    }
                                  },
                                  "loc": {
                                    "start": 8663,
                                    "end": 8804
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "resources",
                                    "loc": {
                                      "start": 8821,
                                      "end": 8830
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
                                            "start": 8853,
                                            "end": 8855
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8853,
                                          "end": 8855
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "index",
                                          "loc": {
                                            "start": 8876,
                                            "end": 8881
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8876,
                                          "end": 8881
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "link",
                                          "loc": {
                                            "start": 8902,
                                            "end": 8906
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8902,
                                          "end": 8906
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "usedFor",
                                          "loc": {
                                            "start": 8927,
                                            "end": 8934
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8927,
                                          "end": 8934
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "translations",
                                          "loc": {
                                            "start": 8955,
                                            "end": 8967
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
                                                  "start": 8994,
                                                  "end": 8996
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 8994,
                                                "end": 8996
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "language",
                                                "loc": {
                                                  "start": 9021,
                                                  "end": 9029
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 9021,
                                                "end": 9029
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "description",
                                                "loc": {
                                                  "start": 9054,
                                                  "end": 9065
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 9054,
                                                "end": 9065
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "name",
                                                "loc": {
                                                  "start": 9090,
                                                  "end": 9094
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 9090,
                                                "end": 9094
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 8968,
                                            "end": 9116
                                          }
                                        },
                                        "loc": {
                                          "start": 8955,
                                          "end": 9116
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 8831,
                                      "end": 9134
                                    }
                                  },
                                  "loc": {
                                    "start": 8821,
                                    "end": 9134
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8599,
                                "end": 9148
                              }
                            },
                            "loc": {
                              "start": 8586,
                              "end": 9148
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "schedule",
                              "loc": {
                                "start": 9161,
                                "end": 9169
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
                                      "start": 9191,
                                      "end": 9206
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 9188,
                                    "end": 9206
                                  }
                                }
                              ],
                              "loc": {
                                "start": 9170,
                                "end": 9220
                              }
                            },
                            "loc": {
                              "start": 9161,
                              "end": 9220
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "id",
                              "loc": {
                                "start": 9233,
                                "end": 9235
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 9233,
                              "end": 9235
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "name",
                              "loc": {
                                "start": 9248,
                                "end": 9252
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 9248,
                              "end": 9252
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "description",
                              "loc": {
                                "start": 9265,
                                "end": 9276
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 9265,
                              "end": 9276
                            }
                          }
                        ],
                        "loc": {
                          "start": 5257,
                          "end": 9286
                        }
                      },
                      "loc": {
                        "start": 5246,
                        "end": 9286
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "handle",
                        "loc": {
                          "start": 9295,
                          "end": 9301
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9295,
                        "end": 9301
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "hasPremium",
                        "loc": {
                          "start": 9310,
                          "end": 9320
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9310,
                        "end": 9320
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "id",
                        "loc": {
                          "start": 9329,
                          "end": 9331
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9329,
                        "end": 9331
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "languages",
                        "loc": {
                          "start": 9340,
                          "end": 9349
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9340,
                        "end": 9349
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "membershipsCount",
                        "loc": {
                          "start": 9358,
                          "end": 9374
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9358,
                        "end": 9374
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "name",
                        "loc": {
                          "start": 9383,
                          "end": 9387
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9383,
                        "end": 9387
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "notesCount",
                        "loc": {
                          "start": 9396,
                          "end": 9406
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9396,
                        "end": 9406
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "projectsCount",
                        "loc": {
                          "start": 9415,
                          "end": 9428
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9415,
                        "end": 9428
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "questionsAskedCount",
                        "loc": {
                          "start": 9437,
                          "end": 9456
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9437,
                        "end": 9456
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "routinesCount",
                        "loc": {
                          "start": 9465,
                          "end": 9478
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9465,
                        "end": 9478
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "smartContractsCount",
                        "loc": {
                          "start": 9487,
                          "end": 9506
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9487,
                        "end": 9506
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "standardsCount",
                        "loc": {
                          "start": 9515,
                          "end": 9529
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9515,
                        "end": 9529
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "theme",
                        "loc": {
                          "start": 9538,
                          "end": 9543
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9538,
                        "end": 9543
                      }
                    }
                  ],
                  "loc": {
                    "start": 396,
                    "end": 9549
                  }
                },
                "loc": {
                  "start": 390,
                  "end": 9549
                }
              }
            ],
            "loc": {
              "start": 356,
              "end": 9553
            }
          },
          "loc": {
            "start": 329,
            "end": 9553
          }
        }
      ],
      "loc": {
        "start": 325,
        "end": 9555
      }
    },
    "loc": {
      "start": 277,
      "end": 9555
    }
  },
  "variableValues": {},
  "path": {
    "key": "auth_emailSignUp"
  }
} as const;
